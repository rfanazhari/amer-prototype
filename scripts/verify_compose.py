"""Verify an isolated local Compose project; credentials never leave the environment."""
import json
import os
from pathlib import Path
import secrets
import subprocess
import sys
import time
import urllib.request

root = str(Path(__file__).resolve().parents[1])
env = os.environ.copy()
secret = secrets.token_hex(20)
env.update(POSTGRES_PASSWORD=secret, RABBITMQ_PASSWORD=secrets.token_hex(20),
           ACCESS_BOOTSTRAP_ADMIN_NAME="Compose Admin",
           ACCESS_BOOTSTRAP_ADMIN_EMAIL="compose-admin@example.com",
           ACCESS_BOOTSTRAP_ADMIN_PASSWORD="correct-horse",
           ACCESS_HTTP_ORIGIN="http://localhost:8080",
           ACCESS_CSRF_SECRET=secrets.token_hex(16),
           POSTGRES_PORT="0", HTTP_PORT="0", REDIS_PORT="0", RABBITMQ_PORT="0",
           RABBITMQ_MANAGEMENT_PORT="0")
project = "amer-prototype-check-" + secrets.token_hex(3)
env["APP_IMAGE"] = project + ":local"
base = ["docker", "compose", "--project-name", project, "--profile", "redis", "--profile", "rabbitmq"]
stack = "--stack" in sys.argv


def run(args, **kwargs):
    """Execute a checked command with isolated project settings and inherited credentials."""
    return subprocess.run(args, cwd=root, env=env, check=True, **kwargs)


def port(service, container_port):
    """Read the ephemeral host port assigned to one service in this verification project."""
    result = run(base + ["port", service, str(container_port)], capture_output=True, text=True)
    return result.stdout.strip().rsplit(":", 1)[1]


try:
    run(base + ["config", "--quiet"])
    if stack:
        run(base + ["up", "--build", "-d", "--wait", "--wait-timeout", "120"])
        url = "http://127.0.0.1:" + port("app", 8080)
        deadline = time.monotonic() + 10
        while True:
            try:
                with urllib.request.urlopen(url + "/health", timeout=2) as response:
                    body = json.load(response)
                    assert body == {"status": "success", "data": {}}
                break
            except (OSError, AssertionError):
                if time.monotonic() >= deadline:
                    raise
                time.sleep(0.1)
        print("Container HTTP health, PostgreSQL, Redis and RabbitMQ readiness passed", flush=True)
    else:
        run(base + ["up", "-d", "--wait", "--wait-timeout", "90", "database"])
    env["TEST_DATABASE_URL"] = (
        f"postgres://amer:{secret}@127.0.0.1:{port('database', 5432)}/amer_prototype?sslmode=disable"
    )
    print("Running real PostgreSQL/API lifecycle checks against isolated Compose database", flush=True)
    run(["go", "test", "-race", "-p", "1", "./test/smoke/...", "./test/integration/...", "-count=1", "-v"])
finally:
    subprocess.run(base + ["down", "--volumes"], cwd=root, env=env, check=True)
