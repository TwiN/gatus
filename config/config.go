package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/TwiN/deepmerge"
	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/announcement"
	"github.com/TwiN/gatus/v5/config/connectivity"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/key"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/config/tunneling"
	"github.com/TwiN/gatus/v5/config/ui"
	"github.com/TwiN/gatus/v5/config/web"
	"github.com/TwiN/gatus/v5/security"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/logr"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultConfigurationFilePath is the default path that will be used to search for the configuration file
	// if a custom path isn't configured through the GATUS_CONFIG_PATH environment variable
	DefaultConfigurationFilePath = "config/config.yaml"

	// DefaultFallbackConfigurationFilePath is the default fallback path that will be used to search for the
	// configuration file if DefaultConfigurationFilePath didn't work
	DefaultFallbackConfigurationFilePath = "config/config.yml"

	// DefaultConcurrency is the default number of endpoints/suites that can be monitored concurrently
	DefaultConcurrency = 3
)

var (
	// ErrNoEndpointOrSuiteInConfig is an error returned when a configuration file or directory has no endpoints configured
	ErrNoEndpointOrSuiteInConfig = errors.New("configuration should contain at least one endpoint or suite")

	// ErrConfigFileNotFound is an error returned when a configuration file could not be found
	ErrConfigFileNotFound = errors.New("configuration file not found")

	// ErrInvalidSecurityConfig is an error returned when the security configuration is invalid
	ErrInvalidSecurityConfig = errors.New("invalid security configuration")

	// errEarlyReturn is returned to break out of a loop from a callback early
	errEarlyReturn = errors.New("early escape")
)

// Config is the main configuration structure
type Config struct {
	// Debug Whether to enable debug logs
	// Deprecated: Use the GATUS_LOG_LEVEL environment variable instead
	Debug bool `yaml:"debug,omitempty"`

	// Metrics Whether to expose metrics at /metrics
	Metrics bool `yaml:"metrics,omitempty"`

	// SkipInvalidConfigUpdate Whether to make the application ignore invalid configuration
	// if the configuration file is updated while the application is running
	SkipInvalidConfigUpdate bool `yaml:"skip-invalid-config-update,omitempty"`

	// DisableMonitoringLock Whether to disable the monitoring lock
	// The monitoring lock is what prevents multiple endpoints from being processed at the same time.
	// Disabling this may lead to inaccurate response times
	//
	// Deprecated: Use Concurrency instead TODO: REMOVE THIS IN v6.0.0
	DisableMonitoringLock bool `yaml:"disable-monitoring-lock,omitempty"`

	// Concurrency is the maximum number of endpoints/suites that can be monitored concurrently
	// Defaults to DefaultConcurrency. Set to 0 for unlimited concurrency.
	Concurrency int `yaml:"concurrency,omitempty"`

	// Security is the configuration for securing access to Gatus
	Security *security.Config `yaml:"security,omitempty"`

	// Alerting is the configuration for alerting providers
	Alerting *alerting.Config `yaml:"alerting,omitempty"`

	// Endpoints is the list of endpoints to monitor
	Endpoints []*endpoint.Endpoint `yaml:"endpoints,omitempty"`

	// ExternalEndpoints is the list of all external endpoints
	ExternalEndpoints []*endpoint.ExternalEndpoint `yaml:"external-endpoints,omitempty"`

	// Suites is the list of suites to monitor
	Suites []*suite.Suite `yaml:"suites,omitempty"`

	// Storage is the configuration for how the data is stored
	Storage *storage.Config `yaml:"storage,omitempty"`

	// Web is the web configuration for the application
	Web *web.Config `yaml:"web,omitempty"`

	// UI is the configuration for the UI
	UI *ui.Config `yaml:"ui,omitempty"`

	// Maintenance is the configuration for creating a maintenance window in which no alerts are sent
	Maintenance *maintenance.Config `yaml:"maintenance,omitempty"`

	// Remote is the configuration for remote Gatus instances
	// WARNING: This is in ALPHA and may change or be completely removed in the future
	Remote *remote.Config `yaml:"remote,omitempty"`

	// Connectivity is the configuration for connectivity
	Connectivity *connectivity.Config `yaml:"connectivity,omitempty"`

	// Tunneling is the configuration for SSH tunneling
	Tunneling *tunneling.Config `yaml:"tunneling,omitempty"`

	// Announcements is the list of system-wide announcements
	Announcements []*announcement.Announcement `yaml:"announcements,omitempty"`

	configPath      string    // path to the file or directory from which config was loaded
	lastFileModTime time.Time // last modification time
}

// GetUniqueExtraMetricLabels returns a slice of unique metric labels from all enabled endpoints
// in the configuration. It iterates through each endpoint, checks if it is enabled,
// and then collects unique labels from the endpoint's labels map.
func (config *Config) GetUniqueExtraMetricLabels() []string {
	labels := make([]string, 0)
	for _, ep := range config.Endpoints {
		if !ep.IsEnabled() {
			continue
		}
		for label := range ep.ExtraLabels {
			if contains(labels, label) {
				continue
			}
			labels = append(labels, label)
		}
	}
	if len(labels) > 1 {
		sort.Strings(labels)
	}
	return labels
}

func (config *Config) GetEndpointByKey(key string) *endpoint.Endpoint {
	for i := 0; i < len(config.Endpoints); i++ {
		ep := config.Endpoints[i]
		if ep.Key() == strings.ToLower(key) {
			return ep
		}
	}
	return nil
}

func (config *Config) GetExternalEndpointByKey(key string) *endpoint.ExternalEndpoint {
	for i := 0; i < len(config.ExternalEndpoints); i++ {
		ee := config.ExternalEndpoints[i]
		if ee.Key() == strings.ToLower(key) {
			return ee
		}
	}
	return nil
}

// HasLoadedConfigurationBeenModified returns whether one of the file that the
// configuration has been loaded from has been modified since it was last read
func (config *Config) HasLoadedConfigurationBeenModified() bool {
	lastMod := config.lastFileModTime.Unix()
	fileInfo, err := os.Stat(config.configPath)
	if err != nil {
		return false
	}
	if fileInfo.IsDir() {
		err = walkConfigDir(config.configPath, func(path string, d fs.DirEntry, err error) error {
			if info, err := d.Info(); err == nil && lastMod < info.ModTime().Unix() {
				return errEarlyReturn
			}
			return nil
		})
		return errors.Is(err, errEarlyReturn)
	}
	return !fileInfo.ModTime().IsZero() && config.lastFileModTime.Unix() < fileInfo.ModTime().Unix()
}

// UpdateLastFileModTime refreshes Config.lastFileModTime
func (config *Config) UpdateLastFileModTime() {
	config.lastFileModTime = time.Now()
}

// LoadConfiguration loads the full configuration composed of the main configuration file
// and all composed configuration files
func LoadConfiguration(configPath string) (*Config, error) {
	var configBytes []byte
	var fileInfo os.FileInfo
	var usedConfigPath string
	// Figure out what config path we'll use (either configPath or the default config path)
	for _, configurationPath := range []string{configPath, DefaultConfigurationFilePath, DefaultFallbackConfigurationFilePath} {
		if len(configurationPath) == 0 {
			continue
		}
		var err error
		fileInfo, err = os.Stat(configurationPath)
		if err != nil {
			continue
		}
		usedConfigPath = configurationPath
		break
	}
	if len(usedConfigPath) == 0 {
		return nil, ErrConfigFileNotFound
	}
	var config *Config
	if fileInfo.IsDir() {
		err := walkConfigDir(configPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("error walking path %s: %w", path, err)
			}
			if strings.Contains(path, "..") {
				logr.Warnf("[config.LoadConfiguration] Ignoring configuration from %s", path)
				return nil
			}
			logr.Infof("[config.LoadConfiguration] Reading configuration from %s", path)
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading configuration from file %s: %w", path, err)
			}
			configBytes, err = deepmerge.YAML(configBytes, data)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("error reading configuration from directory %s: %w", usedConfigPath, err)
		}
	} else {
		logr.Infof("[config.LoadConfiguration] Reading configuration from configFile=%s", usedConfigPath)
		if data, err := os.ReadFile(usedConfigPath); err != nil {
			return nil, fmt.Errorf("error reading configuration from directory %s: %w", usedConfigPath, err)
		} else {
			configBytes = data
		}
	}
	if len(configBytes) == 0 {
		return nil, ErrConfigFileNotFound
	}
	config, err := parseAndValidateConfigBytes(configBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}
	config.configPath = usedConfigPath
	config.UpdateLastFileModTime()
	return config, nil
}

// walkConfigDir is a wrapper for filepath.WalkDir that strips directories and non-config files
func walkConfigDir(path string, fn fs.WalkDirFunc) error {
	if len(path) == 0 {
		// If the user didn't provide a directory, we'll just use the default config file, so we can return nil now.
		return nil
	}
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d == nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		return fn(path, d, err)
	})
}

// parseAndValidateConfigBytes parses a Gatus configuration file into a Config struct and validates its parameters
func parseAndValidateConfigBytes(yamlBytes []byte) (config *Config, err error) {
	// Replace $$ with __GATUS_LITERAL_DOLLAR_SIGN__ to prevent os.ExpandEnv from treating "$$" as if it was an
	// environment variable. This allows Gatus to support literal "$" in the configuration file.
	yamlBytes = []byte(strings.ReplaceAll(string(yamlBytes), "$$", "__GATUS_LITERAL_DOLLAR_SIGN__"))
	// Expand environment variables
	yamlBytes = []byte(os.ExpandEnv(string(yamlBytes)))
	// Replace __GATUS_LITERAL_DOLLAR_SIGN__ with "$" to restore the literal "$" in the configuration file
	yamlBytes = []byte(strings.ReplaceAll(string(yamlBytes), "__GATUS_LITERAL_DOLLAR_SIGN__", "$"))
	// Parse configuration file
	if err = yaml.Unmarshal(yamlBytes, &config); err != nil {
		return
	}
	// Check if the configuration file at least has endpoints configured
	if config == nil || (len(config.Endpoints) == 0 && len(config.Suites) == 0) {
		err = ErrNoEndpointOrSuiteInConfig
	} else {
		// XXX: Remove this in v6.0.0
		if config.Debug {
			logr.Warn("WARNING: The 'debug' configuration has been deprecated and will be removed in v6.0.0")
			logr.Warn("WARNING: Please use the GATUS_LOG_LEVEL environment variable instead")
		}
		// XXX: End of v6.0.0 removals
		ValidateAlertingConfig(config.Alerting, config.Endpoints, config.ExternalEndpoints)
		if err := ValidateSecurityConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateEndpointsConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateWebConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateUIConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateMaintenanceConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateStorageConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateRemoteConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateConnectivityConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateTunnelingConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateAnnouncementsConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateSuitesConfig(config); err != nil {
			return nil, err
		}
		if err := ValidateUniqueKeys(config); err != nil {
			return nil, err
		}
		ValidateAndSetConcurrencyDefaults(config)
		// Cross-config changes
		config.UI.MaximumNumberOfResults = config.Storage.MaximumNumberOfResults
	}
	return
}

func ValidateConnectivityConfig(config *Config) error {
	if config.Connectivity != nil {
		return config.Connectivity.ValidateAndSetDefaults()
	}
	return nil
}

// ValidateTunnelingConfig validates the tunneling configuration and resolves tunnel references
// NOTE: This must be called after ValidateEndpointsConfig and ValidateSuitesConfig
// because it resolves tunnel references in endpoint and suite client configurations
func ValidateTunnelingConfig(config *Config) error {
	if config.Tunneling != nil {
		if err := config.Tunneling.ValidateAndSetDefaults(); err != nil {
			return err
		}
		// Resolve tunnel references in all endpoints
		for _, ep := range config.Endpoints {
			if err := resolveTunnelForClientConfig(config, ep.ClientConfig); err != nil {
				return fmt.Errorf("endpoint '%s': %w", ep.Key(), err)
			}
		}
		// Resolve tunnel references in suite endpoints
		for _, s := range config.Suites {
			for _, ep := range s.Endpoints {
				if err := resolveTunnelForClientConfig(config, ep.ClientConfig); err != nil {
					return fmt.Errorf("suite '%s' endpoint '%s': %w", s.Key(), ep.Key(), err)
				}
			}
		}
		// TODO: Add tunnel support for alert providers when needed
	}
	return nil
}

// resolveTunnelForClientConfig resolves tunnel references in a client configuration
func resolveTunnelForClientConfig(config *Config, clientConfig *client.Config) error {
	if clientConfig == nil || clientConfig.Tunnel == "" {
		return nil
	}
	// Validate tunnel name
	tunnelName := strings.TrimSpace(clientConfig.Tunnel)
	if tunnelName == "" {
		return fmt.Errorf("tunnel name cannot be empty")
	}
	if config.Tunneling == nil {
		return fmt.Errorf("tunnel '%s' referenced but no tunneling configuration defined", tunnelName)
	}
	_, exists := config.Tunneling.Tunnels[tunnelName]
	if !exists {
		return fmt.Errorf("tunnel '%s' not found in tunneling configuration", tunnelName)
	}
	// Get or create the SSH tunnel instance and store it directly in client config
	tunnel, err := config.Tunneling.GetTunnel(tunnelName)
	if err != nil {
		return fmt.Errorf("failed to get tunnel '%s': %w", tunnelName, err)
	}
	clientConfig.ResolvedTunnel = tunnel
	return nil
}

func ValidateAnnouncementsConfig(config *Config) error {
	if config.Announcements != nil {
		if err := announcement.ValidateAndSetDefaults(config.Announcements); err != nil {
			return err
		}
		// Sort announcements by timestamp (newest first) for API response
		announcement.SortByTimestamp(config.Announcements)
	}
	return nil
}

func ValidateRemoteConfig(config *Config) error {
	if config.Remote != nil {
		if err := config.Remote.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}

func ValidateStorageConfig(config *Config) error {
	if config.Storage == nil {
		config.Storage = &storage.Config{
			Type:                   storage.TypeMemory,
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		}
	} else {
		if err := config.Storage.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}

func ValidateMaintenanceConfig(config *Config) error {
	if config.Maintenance == nil {
		config.Maintenance = maintenance.GetDefaultConfig()
	} else {
		if err := config.Maintenance.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}

func ValidateUIConfig(config *Config) error {
	if config.UI == nil {
		config.UI = ui.GetDefaultConfig()
	} else {
		if err := config.UI.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	return nil
}

func ValidateWebConfig(config *Config) error {
	if config.Web == nil {
		config.Web = web.GetDefaultConfig()
	} else {
		return config.Web.ValidateAndSetDefaults()
	}
	return nil
}

func ValidateEndpointsConfig(config *Config) error {
	duplicateValidationMap := make(map[string]bool)
	// Validate endpoints
	for _, ep := range config.Endpoints {
		logr.Debugf("[config.ValidateEndpointsConfig] Validating endpoint with key %s", ep.Key())
		if endpointKey := ep.Key(); duplicateValidationMap[endpointKey] {
			return fmt.Errorf("invalid endpoint %s: name and group combination must be unique", ep.Key())
		} else {
			duplicateValidationMap[endpointKey] = true
		}
		if err := ep.ValidateAndSetDefaults(); err != nil {
			return fmt.Errorf("invalid endpoint %s: %w", ep.Key(), err)
		}
	}
	logr.Infof("[config.ValidateEndpointsConfig] Validated %d endpoints", len(config.Endpoints))
	// Validate external endpoints
	for _, ee := range config.ExternalEndpoints {
		logr.Debugf("[config.ValidateEndpointsConfig] Validating external endpoint '%s'", ee.Key())
		if endpointKey := ee.Key(); duplicateValidationMap[endpointKey] {
			return fmt.Errorf("invalid external endpoint %s: name and group combination must be unique", ee.Key())
		} else {
			duplicateValidationMap[endpointKey] = true
		}
		if err := ee.ValidateAndSetDefaults(); err != nil {
			return fmt.Errorf("invalid external endpoint %s: %w", ee.Key(), err)
		}
	}
	logr.Infof("[config.ValidateEndpointsConfig] Validated %d external endpoints", len(config.ExternalEndpoints))
	return nil
}

func ValidateSuitesConfig(config *Config) error {
	if config.Suites == nil || len(config.Suites) == 0 {
		logr.Info("[config.ValidateSuitesConfig] No suites configured")
		return nil
	}
	suiteNames := make(map[string]bool)
	for _, suite := range config.Suites {
		// Check for duplicate suite names
		if suiteNames[suite.Name] {
			return fmt.Errorf("duplicate suite name: %s", suite.Key())
		}
		suiteNames[suite.Name] = true
		// Validate the suite configuration
		if err := suite.ValidateAndSetDefaults(); err != nil {
			return fmt.Errorf("invalid suite '%s': %w", suite.Key(), err)
		}
		// Check that endpoints referenced in Store mappings use valid placeholders
		for _, suiteEndpoint := range suite.Endpoints {
			if suiteEndpoint.Store != nil {
				for contextKey, placeholder := range suiteEndpoint.Store {
					// Basic validation that the context key is a valid identifier
					if len(contextKey) == 0 {
						return fmt.Errorf("suite '%s' endpoint '%s' has empty context key in store mapping", suite.Key(), suiteEndpoint.Key())
					}
					if len(placeholder) == 0 {
						return fmt.Errorf("suite '%s' endpoint '%s' has empty placeholder in store mapping for key '%s'", suite.Key(), suiteEndpoint.Key(), contextKey)
					}
				}
			}
		}
	}
	logr.Infof("[config.ValidateSuitesConfig] Validated %d suite(s)", len(config.Suites))
	return nil
}

func ValidateUniqueKeys(config *Config) error {
	keyMap := make(map[string]string) // key -> description for error messages
	// Check all endpoints
	for _, ep := range config.Endpoints {
		epKey := ep.Key()
		if existing, exists := keyMap[epKey]; exists {
			return fmt.Errorf("duplicate key '%s': endpoint '%s' conflicts with %s", epKey, ep.Key(), existing)
		}
		keyMap[epKey] = fmt.Sprintf("endpoint '%s'", ep.Key())
	}
	// Check all external endpoints
	for _, ee := range config.ExternalEndpoints {
		eeKey := ee.Key()
		if existing, exists := keyMap[eeKey]; exists {
			return fmt.Errorf("duplicate key '%s': external endpoint '%s' conflicts with %s", eeKey, ee.Key(), existing)
		}
		keyMap[eeKey] = fmt.Sprintf("external endpoint '%s'", ee.Key())
	}
	// Check all suites
	for _, suite := range config.Suites {
		suiteKey := suite.Key()
		if existing, exists := keyMap[suiteKey]; exists {
			return fmt.Errorf("duplicate key '%s': suite '%s' conflicts with %s", suiteKey, suite.Key(), existing)
		}
		keyMap[suiteKey] = fmt.Sprintf("suite '%s'", suite.Key())
		// Check endpoints within suites (they generate keys using suite group + endpoint name)
		for _, ep := range suite.Endpoints {
			epKey := key.ConvertGroupAndNameToKey(suite.Group, ep.Name)
			if existing, exists := keyMap[epKey]; exists {
				return fmt.Errorf("duplicate key '%s': endpoint '%s' in suite '%s' conflicts with %s", epKey, epKey, suite.Key(), existing)
			}
			keyMap[epKey] = fmt.Sprintf("endpoint '%s' in suite '%s'", epKey, suite.Key())
		}
	}
	return nil
}

func ValidateSecurityConfig(config *Config) error {
	if config.Security != nil {
		if !config.Security.ValidateAndSetDefaults() {
			logr.Debug("[config.ValidateSecurityConfig] Basic security configuration has been validated")
			return ErrInvalidSecurityConfig
		}
	}
	return nil
}

// ValidateAlertingConfig validates the alerting configuration
// Note that the alerting configuration has to be validated before the endpoint configuration, because the default alert
// returned by provider.AlertProvider.GetDefaultAlert() must be parsed before endpoint.Endpoint.ValidateAndSetDefaults()
// sets the default alert values when none are set.
func ValidateAlertingConfig(alertingConfig *alerting.Config, endpoints []*endpoint.Endpoint, externalEndpoints []*endpoint.ExternalEndpoint) {
	if alertingConfig == nil {
		logr.Info("[config.ValidateAlertingConfig] Alerting is not configured")
		return
	}
	alertTypes := []alert.Type{
		alert.TypeAWSSES,
		alert.TypeClickUp,
		alert.TypeCustom,
		alert.TypeDatadog,
		alert.TypeDiscord,
		alert.TypeEmail,
		alert.TypeGitHub,
		alert.TypeGitLab,
		alert.TypeGitea,
		alert.TypeGoogleChat,
		alert.TypeGotify,
		alert.TypeHomeAssistant,
		alert.TypeIFTTT,
		alert.TypeIlert,
		alert.TypeIncidentIO,
		alert.TypeLine,
		alert.TypeMatrix,
		alert.TypeMattermost,
		alert.TypeMessagebird,
		alert.TypeN8N,
		alert.TypeNewRelic,
		alert.TypeNtfy,
		alert.TypeOpsgenie,
		alert.TypePagerDuty,
		alert.TypePlivo,
		alert.TypePushover,
		alert.TypeRocketChat,
		alert.TypeSendGrid,
		alert.TypeSignal,
		alert.TypeSIGNL4,
		alert.TypeSlack,
		alert.TypeSplunk,
		alert.TypeSquadcast,
		alert.TypeTeams,
		alert.TypeTeamsWorkflows,
		alert.TypeTelegram,
		alert.TypeTwilio,
		alert.TypeVonage,
		alert.TypeWebex,
		alert.TypeZapier,
		alert.TypeZulip,
	}
	var validProviders, invalidProviders []alert.Type
	for _, alertType := range alertTypes {
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(alertType)
		if alertProvider != nil {
			if err := alertProvider.Validate(); err == nil {
				// Parse alerts with the provider's default alert
				if alertProvider.GetDefaultAlert() != nil {
					for _, ep := range endpoints {
						for alertIndex, endpointAlert := range ep.Alerts {
							if alertType == endpointAlert.Type {
								logr.Debugf("[config.ValidateAlertingConfig] Parsing alert %d with default alert for provider=%s in endpoint with key=%s", alertIndex, alertType, ep.Key())
								provider.MergeProviderDefaultAlertIntoEndpointAlert(alertProvider.GetDefaultAlert(), endpointAlert)
								// Validate the endpoint alert's overrides, if applicable
								if len(endpointAlert.ProviderOverride) > 0 {
									if err = alertProvider.ValidateOverrides(ep.Group, endpointAlert); err != nil {
										logr.Warnf("[config.ValidateAlertingConfig] endpoint with key=%s has invalid overrides for provider=%s: %s", ep.Key(), alertType, err.Error())
									}
								}
							}
						}
					}
					for _, ee := range externalEndpoints {
						for alertIndex, endpointAlert := range ee.Alerts {
							if alertType == endpointAlert.Type {
								logr.Debugf("[config.ValidateAlertingConfig] Parsing alert %d with default alert for provider=%s in endpoint with key=%s", alertIndex, alertType, ee.Key())
								provider.MergeProviderDefaultAlertIntoEndpointAlert(alertProvider.GetDefaultAlert(), endpointAlert)
								// Validate the endpoint alert's overrides, if applicable
								if len(endpointAlert.ProviderOverride) > 0 {
									if err = alertProvider.ValidateOverrides(ee.Group, endpointAlert); err != nil {
										logr.Warnf("[config.ValidateAlertingConfig] endpoint with key=%s has invalid overrides for provider=%s: %s", ee.Key(), alertType, err.Error())
									}
								}
							}
						}
					}
				}
				validProviders = append(validProviders, alertType)
			} else {
				logr.Warnf("[config.ValidateAlertingConfig] Ignoring provider=%s due to error=%s", alertType, err.Error())
				invalidProviders = append(invalidProviders, alertType)
				alertingConfig.SetAlertingProviderToNil(alertProvider)
			}
		} else {
			invalidProviders = append(invalidProviders, alertType)
		}
	}
	logr.Infof("[config.ValidateAlertingConfig] configuredProviders=%s; ignoredProviders=%s", validProviders, invalidProviders)
}

func ValidateAndSetConcurrencyDefaults(config *Config) {
	if config.DisableMonitoringLock {
		config.Concurrency = 0
		logr.Warn("WARNING: The 'disable-monitoring-lock' configuration has been deprecated and will be removed in v6.0.0")
		logr.Warn("WARNING: Please set 'concurrency: 0' instead")
		logr.Debug("[config.ValidateAndSetConcurrencyDefaults] DisableMonitoringLock is true, setting unlimited (0) concurrency")
	} else if config.Concurrency <= 0 && !config.DisableMonitoringLock {
		config.Concurrency = DefaultConcurrency
		logr.Debugf("[config.ValidateAndSetConcurrencyDefaults] Setting default concurrency to %d", config.Concurrency)
	} else {
		logr.Debugf("[config.ValidateAndSetConcurrencyDefaults] Using configured concurrency of %d", config.Concurrency)
	}
}
