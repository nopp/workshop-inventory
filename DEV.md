# Workshop Inventory - Desenvolvimento

## Quick Start

Para rodar a aplicação rapidamente em ambiente de desenvolvimento:

```bash
# Setup completo (inicializa dados e sobe o ambiente)
make setup

# Ou paso a passo:
make init-data  # Cria arquivos de dados se não existirem
make dev        # Sobe o ambiente de desenvolvimento
```

## Comandos Disponíveis

```bash
make help       # Lista todos os comandos disponíveis
make dev        # Inicia ambiente de desenvolvimento
make debug      # Inicia com ferramentas de debug (file browser)
make logs       # Mostra logs da aplicação
make restart    # Reinicia a aplicação
make down       # Para todos os serviços
make clean      # Limpa containers e volumes
make shell      # Acessa o shell do container
```

## URLs

- **Aplicação Principal**: http://localhost:9090
- **File Browser (debug)**: http://localhost:9091 (apenas com `make debug`)

## Estrutura de Dados

A aplicação usa arquivos JSON para persistência:

- `dados.json`: Items do inventário
- `usuarios.json`: Usuários do sistema
- `config.json`: Configurações da aplicação
- `static/photos/`: Fotos dos items (com thumbnails em `thumbs/`)

## Desenvolvimento

Os arquivos são montados como volumes, então mudanças em templates e dados são refletidas imediatamente. Para mudanças no código Go, use `make restart` para recompilar.

## Troubleshooting

- **Erro de permissão nas fotos**: `chmod -R 755 static/`
- **Aplicação não inicia**: Verifique se a porta 9090 está livre
- **Ver logs detalhados**: `make logs`
- **Reset completo**: `make clean && make setup`