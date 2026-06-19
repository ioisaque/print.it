# print.it

Microserviço local de impressão para **Bematech MP-4200 TH ADV** (e outras térmicas ESC/POS) via **rede (Ethernet, porta 9100)** no macOS.

Seu app web chama `http://127.0.0.1:9280` e o serviço envia os comandos para a impressora — sem diálogo de impressão.

## Pré-requisitos

1. **Go 1.23+** — instale com Homebrew:

```bash
brew install go
```

2. **Impressora na rede** com IP fixo (ex: `192.168.1.201`)
3. Impressora configurada em modo **ESC/POS** (auto-teste com botão FEED ao ligar → "Conjunto de Comandos")

## Instalação rápida (3 passos)

```bash
cd /Users/ioisaque/Projects/print.it
chmod +x scripts/*.sh
./scripts/build.sh
```

Edite o IP da impressora em `config.json`:

```json
{
  "printer_host": "192.168.1.201",
  "printer_port": 9100
}
```

Inicie o serviço:

```bash
./print.it
```

### Iniciar automaticamente no login (opcional)

```bash
./scripts/install-macos.sh
```

## Interface web

Com o serviço rodando, abra no navegador:

```
http://127.0.0.1:9280
```

A UI permite buscar impressoras na rede, configurar opções, imprimir teste, texto, PDF, código de barras, QR Code e imagem.

## Testar em casa

Com o serviço rodando:

```bash
# 1. Verificar se está no ar
curl http://127.0.0.1:9280/health

# 2. Buscar impressoras na rede (leva alguns segundos)
curl http://127.0.0.1:9280/discover

# 3. Aplicar automaticamente se achar só uma impressora
curl "http://127.0.0.1:9280/discover?auto=true"

# 4. Imprimir página de teste
curl -X POST http://127.0.0.1:9280/print/test

# 5. Imprimir texto
curl -X POST http://127.0.0.1:9280/print \
  -H "Content-Type: application/json" \
  -d '{"text":"Pedido #42\nPizza Calabresa\n","cut":true,"align":"center"}'

# 6. Imprimir PDF
curl -X POST http://127.0.0.1:9280/print/pdf \
  -F "file=@nota.pdf" \
  -F "cut=true"
```

## Integração com app web

```javascript
await fetch("http://127.0.0.1:9280/print", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    text: "Pedido #42\nItem 1\nItem 2\n",
    cut: true,
    align: "left",
    bold: false,
  }),
});
```

Para PDF (multipart):

```javascript
const form = new FormData();
form.append("file", pdfFile);
form.append("cut", "true");

await fetch("http://127.0.0.1:9280/print/pdf", {
  method: "POST",
  body: form,
});
```

> **HTTPS:** se seu app web usa `https://`, o navegador pode bloquear chamadas para `http://127.0.0.1`. Nesse caso use o app em HTTP na rede local ou faça proxy pelo backend na mesma máquina.

## API

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/discover` | Busca dispositivos com porta 9100 aberta na rede local |
| POST | `/discover/apply` | Salva `printer_host`/`printer_port` no `config.json` |
| GET | `/health` | Status do serviço |
| GET | `/config` | Configuração atual |
| PUT | `/config` | Atualizar IP/porta da impressora |
| POST | `/print` | Texto (`json`: `text`, `cut`, `align`, `bold`) |
| POST | `/print/test` | Página de teste |
| POST | `/print/pdf` | PDF (`cut`, `cut_between_pages`, `trim_trailing_blank`) |
| POST | `/print/image` | PNG/JPG (`multipart file` ou `image_base64`) |
| POST | `/print/raw` | Bytes ESC/POS crus (`data_base64` ou `file`) |

## Configuração (`config.json`)

| Campo | Padrão | Descrição |
|-------|--------|-----------|
| `printer_host` | `192.168.1.201` | IP da impressora |
| `printer_port` | `9100` | Porta raw |
| `listen_host` | `127.0.0.1` | Só aceita conexões locais |
| `listen_port` | `9280` | Porta do microserviço |
| `paper_width_mm` | `80` | Largura física do papel (58 ou 80) |
| `printable_width_mm` | `72` em papel 80mm | Área realmente imprimível (evita corte lateral) |
| `trim_trailing_blank` | `false` | Padrão global para recortar branco no fim |
| `cors_origins` | `["*"]` | Origens permitidas no CORS |

## Solução de problemas

| Problema | O que verificar |
|----------|-----------------|
| `connection refused` na impressora | IP correto? Impressora ligada e na mesma rede? |
| Caracteres estranhos | Modo ESC/POS na impressora (não BEMATECH puro para texto) |
| App web não imprime | CORS? Mixed content HTTPS→HTTP? |
| PDF borrado | Normal em térmica — texto pequeno pode perder nitidez |

## Licença

MIT
