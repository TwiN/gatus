# Ambiente exploratório da PTOSS-3

Na branch `ptoss3` do fork, abra o terminal na raiz do clone e execute com o Docker Desktop aberto:

```powershell
Set-Location -LiteralPath '.\docs\ptoss3\ambiente'
docker compose up -d --build
```

Abra `http://127.0.0.1:8080`. O painel apresenta seis serviços, incluindo `painel-falha`, com falha intencional. Se a porta 8080 estiver ocupada, defina `$env:PTOSS3_PORT = '18080'` antes de subir e acesse `http://127.0.0.1:18080`.

Siga o plano e o modelo recebidos da equipe. Use a pasta `docs/ptoss3/registros/EX0X` do seu identificador para o registro e as evidências. Ao concluir, restaure os arquivos de configuração e de apoio usados na sessão e execute `docker compose stop` na pasta `ambiente`.
