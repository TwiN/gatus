package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

type Migrator struct {
	migrator.Migrator
}

type Column struct {
	name              string
	nullable          sql.NullString
	datatype          string
	maxlen            sql.NullInt64
	precision         sql.NullInt64
	radix             sql.NullInt64
	scale             sql.NullInt64
	datetimeprecision sql.NullInt64
}

func (c Column) Name() string {
	return c.name
}

func (c Column) DatabaseTypeName() string {
	return c.datatype
}

func (c Column) Length() (length int64, ok bool) {
	ok = c.maxlen.Valid
	if ok {
		length = c.maxlen.Int64
	} else {
		length = 0
	}
	return
}

func (c Column) Nullable() (nullable bool, ok bool) {
	if c.nullable.Valid {
		nullable, ok = c.nullable.String == "YES", true
	} else {
		nullable, ok = false, false
	}
	return
}

func (c Column) DecimalSize() (precision int64, scale int64, ok bool) {
	if ok = c.precision.Valid && c.scale.Valid && c.radix.Valid && c.radix.Int64 == 10; ok {
		precision, scale = c.precision.Int64, c.scale.Int64
	} else if ok = c.datetimeprecision.Valid; ok {
		precision, scale = c.datetimeprecision.Int64, 0
	} else {
		precision, scale, ok = 0, 0, false
	}
	return
}

func (m Migrator) CurrentDatabase() (name string) {
	m.DB.Raw("SELECT CURRENT_DATABASE()").Row().Scan(&name)
	return
}

func (m Migrator) BuildIndexOptions(opts []schema.IndexOption, stmt *gorm.Statement) (results []interface{}) {
	for _, opt := range opts {
		str := stmt.Quote(opt.DBName)
		if opt.Expression != "" {
			str = opt.Expression
		}

		if opt.Collate != "" {
			str += " COLLATE " + opt.Collate
		}

		if opt.Sort != "" {
			str += " " + opt.Sort
		}
		results = append(results, clause.Expr{SQL: str})
	}
	return
}

func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}

		return m.DB.Raw(
			"SELECT count(*) FROM pg_indexes WHERE tablename = ? AND indexname = ? AND schemaname = ?", stmt.Table, name, m.CurrentSchema(stmt),
		).Row().Scan(&count)
	})

	return count > 0
}

func (m Migrator) CreateIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			opts := m.BuildIndexOptions(idx.Fields, stmt)
			values := []interface{}{clause.Column{Name: idx.Name}, m.CurrentTable(stmt), opts}

			createIndexSQL := "CREATE "
			if idx.Class != "" {
				createIndexSQL += idx.Class + " "
			}
			createIndexSQL += "INDEX ?"

			createIndexSQL += " ON ?"

			if idx.Type != "" {
				createIndexSQL += " USING " + idx.Type + "(?)"
			} else {
				createIndexSQL += " ?"
			}

			if idx.Where != "" {
				createIndexSQL += " WHERE " + idx.Where
			}

			return m.DB.Exec(createIndexSQL, values...).Error
		}

		return fmt.Errorf("failed to create index with name %v", name)
	})
}

func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Exec(
			"ALTER INDEX ? RENAME TO ?",
			clause.Column{Name: oldName}, clause.Column{Name: newName},
		).Error
	})
}

func (m Migrator) DropIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}

		return m.DB.Exec("DROP INDEX ?", clause.Column{Name: name}).Error
	})
}

func (m Migrator) CreateTable(values ...interface{}) (err error) {
	if err = m.Migrator.CreateTable(values...); err != nil {
		return
	}
	for _, value := range m.ReorderModels(values, false) {
		if err = m.RunWithValue(value, func(stmt *gorm.Statement) error {
			for _, field := range stmt.Schema.FieldsByDBName {
				if field.Comment != "" {
					if err := m.DB.Exec(
						"COMMENT ON COLUMN ?.? IS ?",
						m.CurrentTable(stmt), clause.Column{Name: field.DBName}, gorm.Expr(m.Migrator.Dialector.Explain("$1", field.Comment)),
					).Error; err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return
		}
	}
	return
}

func (m Migrator) HasTable(value interface{}) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ? AND table_type = ?", m.CurrentSchema(stmt), stmt.Table, "BASE TABLE").Row().Scan(&count)
	})

	return count > 0
}

func (m Migrator) DropTable(values ...interface{}) error {
	values = m.ReorderModels(values, false)
	tx := m.DB.Session(&gorm.Session{})
	for i := len(values) - 1; i >= 0; i-- {
		if err := m.RunWithValue(values[i], func(stmt *gorm.Statement) error {
			return tx.Exec("DROP TABLE IF EXISTS ? CASCADE", m.CurrentTable(stmt)).Error
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m Migrator) AddColumn(value interface{}, field string) error {
	if err := m.Migrator.AddColumn(value, field); err != nil {
		return err
	}
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			if field.Comment != "" {
				if err := m.DB.Exec(
					"COMMENT ON COLUMN ?.? IS ?",
					m.CurrentTable(stmt), clause.Column{Name: field.DBName}, gorm.Expr(m.Migrator.Dialector.Explain("$1", field.Comment)),
				).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (m Migrator) HasColumn(value interface{}, field string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		name := field
		if field := stmt.Schema.LookUpField(field); field != nil {
			name = field.DBName
		}

		return m.DB.Raw(
			"SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?",
			m.CurrentSchema(stmt), stmt.Table, name,
		).Row().Scan(&count)
	})

	return count > 0
}

func (m Migrator) MigrateColumn(value interface{}, field *schema.Field, columnType gorm.ColumnType) error {
	if err := m.Migrator.MigrateColumn(value, field, columnType); err != nil {
		return err
	}
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		var description string
		values := []interface{}{stmt.Table, field.DBName, stmt.Table, m.CurrentSchema(stmt)}
		checkSQL := "SELECT description FROM pg_catalog.pg_description "
		checkSQL += "WHERE objsubid = (SELECT ordinal_position FROM information_schema.columns WHERE table_name = ? AND column_name = ?) "
		checkSQL += "AND objoid = (SELECT oid FROM pg_catalog.pg_class WHERE relname = ? AND relnamespace = "
		checkSQL += "(SELECT oid FROM pg_catalog.pg_namespace WHERE nspname = ?))"
		m.DB.Raw(checkSQL, values...).Scan(&description)
		comment := field.Comment
		if comment != "" {
			comment = comment[1 : len(comment)-1]
		}
		if field.Comment != "" && comment != description {
			if err := m.DB.Exec(
				"COMMENT ON COLUMN ?.? IS ?",
				m.CurrentTable(stmt), clause.Column{Name: field.DBName}, gorm.Expr(m.Migrator.Dialector.Explain("$1", field.Comment)),
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (m Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw(
			"SELECT count(*) FROM INFORMATION_SCHEMA.table_constraints WHERE table_schema = ? AND table_name = ? AND constraint_name = ?",
			m.CurrentSchema(stmt), stmt.Table, name,
		).Row().Scan(&count)
	})

	return count > 0
}

func (m Migrator) ColumnTypes(value interface{}) (columnTypes []gorm.ColumnType, err error) {
	columnTypes = make([]gorm.ColumnType, 0)
	err = m.RunWithValue(value, func(stmt *gorm.Statement) error {
		currentDatabase := m.DB.Migrator().CurrentDatabase()
		currentSchema := m.CurrentSchema(stmt)
		columns, err := m.DB.Raw(
			"SELECT column_name, is_nullable, udt_name, character_maximum_length, "+
				"numeric_precision, numeric_precision_radix, numeric_scale, datetime_precision "+
				"FROM information_schema.columns WHERE table_catalog = ? AND table_schema = ? AND table_name = ?",
			currentDatabase, currentSchema, stmt.Table).Rows()
		if err != nil {
			return err
		}
		defer columns.Close()

		for columns.Next() {
			var column Column
			err = columns.Scan(
				&column.name,
				&column.nullable,
				&column.datatype,
				&column.maxlen,
				&column.precision,
				&column.radix,
				&column.scale,
				&column.datetimeprecision,
			)
			if err != nil {
				return err
			}
			columnTypes = append(columnTypes, column)
		}

		return err
	})
	return
}

func (m Migrator) CurrentSchema(stmt *gorm.Statement) interface{} {
	if stmt.TableExpr != nil {
		if tables := strings.Split(stmt.TableExpr.SQL, `"."`); len(tables) == 2 {
			return strings.TrimPrefix(tables[0], `"`)
		}
	}
	return clause.Expr{SQL: "CURRENT_SCHEMA()"}
}
