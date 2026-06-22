<picture>
  <source media="(prefers-color-scheme: dark)" srcset="web/assets/imgs/logo-light.jpeg">
  <source media="(prefers-color-scheme: light)" srcset="web/assets/imgs/logo-dark.jpeg">
  <img alt="print.it — agente local de impressão térmica" src="web/assets/imgs/logo-dark.jpeg" height="72">
</picture>

Agente local de impressão térmica **ESC/POS** para Windows, macOS e Linux. Instale uma vez no computador do caixa ou cozinha; o serviço sobe automaticamente no login e imprime via rede (porta 9100) — sem depender do navegador nem de `window.print()`.

Compatível com impressoras térmicas comuns (Bematech MP-4200 TH ADV, Elgin, Epson TM-T20 e similares).

## Instalação

Baixe o instalador da sua plataforma em **[Releases](https://github.com/ioisaque/print.it/releases)**:

| Plataforma | Arquivo | Como instalar |
|------------|---------|---------------|
| **macOS** | `print.it-*-macos-arm64.pkg` ou `*-macos-amd64.pkg` | Duplo clique no `.pkg`, seguir o assistente |
| **Windows** | `print.it-*-windows-amd64.exe` | Executar o instalador (Next, Next…) |
| **Linux** | `print.it-*-linux-amd64.deb` | `sudo dpkg -i print.it-*.deb` |

Requisito Linux: **glibc 2.35+** (Ubuntu 22.04, Debian 12 ou mais recente). Verifique com `ldd --version`.

Depois da instalação, faça **login** (ou reinicie no macOS). O agente inicia sozinho em segundo plano.

## Como usar

1. Abra a interface local no navegador: **[http://127.0.0.1:9280/printit/](http://127.0.0.1:9280/printit/)**
2. Toque em **Conectar impressora** → **Buscar na rede** e selecione a impressora.
3. Ajuste largura do papel (58 ou 80 mm) e outras opções, se precisar.
4. Use as abas da interface para imprimir **texto**, **PDF**, **código de barras**, **QR Code** ou **imagem** — ou deixe seu sistema de gestão (PDV, ERP, etc.) enviar os trabalhos pela API HTTP.

A interface web serve para configuração manual e testes. No dia a dia, o seu software pode imprimir direto em `http://127.0.0.1:9280/printit/` sem abrir o navegador.

### Desinstalar

```bash
print.it --uninstall
```

No Windows, use também **Adicionar ou remover programas**.

## Integração (PDV / sistemas)

Se você desenvolve ou integra um sistema de restaurantes, ERP ou PDV:

1. Verifique se o agente está online: `GET http://127.0.0.1:9280/printit/health`
2. Descubra impressoras: `GET /printit/discover`
3. Configure remotamente: `PUT /printit/config` (IP, porta, largura do papel)
4. Imprima: `POST /printit/text`, `/pdf`, `/barcode`, `/qrcode`, `/image`

Exemplo rápido:

```javascript
const PRINTIT = "http://127.0.0.1:9280/printit";

await fetch(`${PRINTIT}/text`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ text: "Pedido #42\nPizza Calabresa\n", cut: true }),
});
```

> Mantenha `listen_host` em `127.0.0.1` — o agente é local e não deve ficar exposto na rede.

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/printit/health` | Agente online? (`ok`, `version`) |
| GET | `/printit/discover` | Busca impressoras na rede |
| PUT | `/printit/config` | IP, porta, papel |
| POST | `/printit/text` | Texto |
| POST | `/printit/pdf` | PDF (multipart) |
| POST | `/printit/barcode` | Código de barras |
| POST | `/printit/qrcode` | QR Code |
| POST | `/printit/image` | Imagem |
| POST | `/printit/test` | Página de teste na impressora |

## Desenvolvimento

Requisitos: Go 1.24+, impressora térmica na rede.

```bash
git clone https://github.com/ioisaque/print.it.git
cd print.it
chmod +x scripts/*.sh
./scripts/build.sh
./print.it
```

Config e logs ficam em `~/Library/Application Support/print.it/` (macOS), `~/.config/print.it/` (Linux) ou `%AppData%\print.it\` (Windows).

Build de release: `./scripts/build-all.sh` · CI publica instaladores ao criar tag `v*`.

## Licença

MIT
