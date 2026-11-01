#!/usr/bin/env python3
"""Exercise the real candidate page script in isolated headless Chrome with API fixtures."""

import http.server
import json
import os
from pathlib import Path
import subprocess
import tempfile
import threading

ROOT = Path(__file__).resolve().parents[1]


class FixtureHandler(http.server.BaseHTTPRequestHandler):
    """Serve only a browser harness containing the current page, without application resources."""

    def do_GET(self):
        """Embed the actual page into the fixture harness and return it."""
        page = (ROOT / 'delivery/http/candidateinterview/page.html').read_text()
        harness = (ROOT / 'scripts/testdata/candidate_interview_ui.html').read_text()
        payload = harness.replace('PAGE_SOURCE', json.dumps(page).replace('</', '<\\/')).encode()
        self.send_response(200)
        self.send_header('Content-Type', 'text/html; charset=utf-8')
        self.end_headers()
        self.wfile.write(payload)

    def log_message(self, *_args):
        """Keep fixture access logs out of the result."""


def main():
    """Run a bounded browser session and fail unless every behavioral scenario passes."""
    chrome = os.environ.get('CHROME_BIN', '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome')
    with http.server.ThreadingHTTPServer(('127.0.0.1', 0), FixtureHandler) as server:
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        try:
            with tempfile.TemporaryDirectory(prefix='core-cin-010-') as profile:
                result = subprocess.run([
                    chrome, '--headless', '--disable-gpu', '--no-first-run',
                    '--no-default-browser-check', '--disable-background-networking',
                    f'--user-data-dir={profile}', '--virtual-time-budget=10000', '--dump-dom',
                    f'http://127.0.0.1:{server.server_port}/',
                ], capture_output=True, text=True, timeout=45)
                # The result marker occurs only after the asynchronous scenarios finish.
                marker = '<pre id="result">'
                output = result.stdout.split(marker)[-1].split('</pre>')[0]
                if marker not in result.stdout or not output.startswith('PASS '):
                    raise SystemExit('FAIL browser scenarios: ' + output[:3000] +
                                     ('\n' + result.stderr[-1000:] if result.returncode else ''))
                print(output)
        finally:
            server.shutdown()
            thread.join(timeout=5)


if __name__ == '__main__':
    main()
