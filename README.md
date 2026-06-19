# print.it

Microserviço local de impressão para **Bematech MP-4200 TH ADV** (e outras térmicas ESC/POS) via **rede (Ethernet, porta 9100)** no macOS.

Seu app web chama `http://127.0.0.1:9280/printit/...` e o serviço envia os comandos para a impressora — sem diálogo de impressão.

## Pré-requisitos

1. **Go 1.23+** — instale com Homebrew:

```bash
brew install go
```

2. **Impressora na rede** com IP fixo (ex: `192.168.1.201`)
3. Impressora configurada em modo **ESC/POS** (auto-teste com botão FEED ao ligar → "Conjunto de Comandos")

## Instalação rápida

```bash
cd /Users/ioisaque/Projects/print.it
chmod +x scripts/*.sh
./scripts/build.sh
```

Edite o IP da impressora em `config.json` ou use a interface web.

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
http://127.0.0.1:9280/printit/
```

A UI permite buscar impressoras na rede, configurar opções, imprimir teste, texto, PDF, código de barras, QR Code e imagem.

Frontend em [`web/`](/Users/ioisaque/Projects/print.it/web/):

```
web/
  index.html
  css/          # variables, base, layout, components
  js/           # api, ui, app + módulos por aba
  assets/imgs/  # logos IdeYou
```

## Testar

```bash
# Status completo (com configuração)
curl http://127.0.0.1:9280/printit/status

# Buscar impressoras na rede
curl http://127.0.0.1:9280/printit/discover

# Página de teste
curl -X POST http://127.0.0.1:9280/printit/test

# Texto
curl -X POST http://127.0.0.1:9280/printit/text \
  -H "Content-Type: application/json" \
  -d '{"text":"Pedido #42\nPizza Calabresa\n","cut":true,"align":"center"}'

# PDF
curl -X POST http://127.0.0.1:9280/printit/pdf \
  -F "file=@nota.pdf" \
  -F "cut=true" \
  -F "cut_between_pages=true" \
  -F "trim_trailing_blank=true"
```

## Integração com app web

```javascript
const PRINTIT = "http://127.0.0.1:9280/printit";

await fetch(`${PRINTIT}/text`, {
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

PDF (multipart):

```javascript
const form = new FormData();
form.append("file", pdfFile);
form.append("cut", "true");
form.append("cut_between_pages", "true");
form.append("trim_trailing_blank", "true");

await fetch(`${PRINTIT}/pdf`, { method: "POST", body: form });
```

> **HTTPS:** se seu app web usa `https://`, o navegador pode bloquear chamadas para `http://127.0.0.1`. Use HTTP na rede local ou proxy pelo backend na mesma máquina.

## API

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/printit` | Info do serviço (`status`, `admin`) |
| GET | `/printit/` | Interface web |
| GET | `/printit/status` | Status do serviço (completo c/ configuração) |
| PUT | `/printit/config` | Atualizar configuração |
| GET | `/printit/discover` | Busca impressoras na rede (porta 9100) |
| POST | `/printit/text` | Imprimir texto |
| POST | `/printit/pdf` | Imprimir PDF |
| POST | `/printit/image` | Imprimir imagem |
| POST | `/printit/raw` | Bytes ESC/POS crus |
| POST | `/printit/barcode` | Código de barras |
| POST | `/printit/qrcode` | QR Code |
| POST | `/printit/test` | Página de teste |

## Configuração (`config.json`)

| Campo | Padrão | Descrição |
|-------|--------|-----------|
| `printer_host` | `192.168.1.201` | IP da impressora |
| `printer_port` | `9100` | Porta raw |
| `listen_host` | `127.0.0.1` | Só aceita conexões locais |
| `listen_port` | `9280` | Porta do microserviço |
| `paper_width_mm` | `80` | Largura física do papel (58 ou 80) |
| `printable_width_mm` | `72` em papel 80mm | Área imprimível (evita corte lateral) |
| `trim_trailing_blank` | `false` | Recortar branco no fim por padrão |
| `cors_origins` | `["*"]` | Origens permitidas no CORS |

## Solução de problemas

| Problema | O que verificar |
|----------|-----------------|
| `connection refused` na impressora | IP correto? Impressora ligada e na mesma rede? |
| Caracteres estranhos | Modo ESC/POS na impressora |
| App web não imprime | CORS? Mixed content HTTPS→HTTP? |
| PDF borrado | Normal em térmica — texto pequeno pode perder nitidez |

## Licença

MIT
