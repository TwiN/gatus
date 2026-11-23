"""Servidor local com respostas controladas para as sessões exploratórias."""

from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path


DADOS = Path(__file__).resolve().parent


class ServidorDeApoio(BaseHTTPRequestHandler):
    def do_GET(self):
        status, corpo, tipo = self.resposta()
        self.send_response(status)
        self.send_header("Content-Type", tipo)
        self.send_header("Content-Length", str(len(corpo)))
        self.end_headers()
        self.wfile.write(corpo)

    def resposta(self):
        texto = "text/plain; charset=utf-8"
        if self.path == "/ok":
            return 200, b"OK", texto
        if self.path == "/falha":
            return 503, b"Indisponivel", texto
        if self.path == "/estado":
            status = (DADOS / "estado.txt").read_text(encoding="utf-8-sig").strip()
            if status in {"200", "201", "503"}:
                return int(status), status.encode("ascii"), texto
            return 500, b"Estado fora do conjunto configurado", texto
        if self.path == "/json":
            corpo = (DADOS / "corpo.json.txt").read_text(encoding="utf-8-sig").encode("utf-8")
            return 200, corpo, "application/json"
        return 404, b"Rota desconhecida", texto


if __name__ == "__main__":
    ThreadingHTTPServer(("0.0.0.0", 8000), ServidorDeApoio).serve_forever()
