"""Prove report dispatch against disposable Compose PostgreSQL and RabbitMQ, without secret output."""
import os
from pathlib import Path
import secrets
import subprocess
import tempfile
import time

ROOT = Path(__file__).resolve().parents[1]


def main():
    """Own one isolated project, stop only its broker, and remove only its resources."""
    env = os.environ.copy()
    project = "irp-proof-" + secrets.token_hex(4)
    password = secrets.token_hex(20)
    broker_password = secrets.token_hex(20)
    env.update(POSTGRES_PASSWORD=password, RABBITMQ_PASSWORD=broker_password,
               POSTGRES_PORT="0", RABBITMQ_PORT="0", RABBITMQ_MANAGEMENT_PORT="0",
               ACCESS_BOOTSTRAP_ADMIN_NAME="Proof", ACCESS_BOOTSTRAP_ADMIN_EMAIL="proof@example.test",
               ACCESS_BOOTSTRAP_ADMIN_PASSWORD=secrets.token_hex(20), ACCESS_CSRF_SECRET=secrets.token_hex(32))
    # Never accept inherited database or broker targets for this destructive outage proof.
    base = ["docker", "compose", "--project-name", project, "--profile", "rabbitmq"]

    def run(args, **kwargs):
        """Run a bounded command without echoing environment credentials."""
        return subprocess.run(args, cwd=ROOT, env=env, check=True, timeout=180, **kwargs)

    def port(service, number):
        """Resolve an ephemeral loopback port without rendering the URL."""
        result = run(base + ["port", service, str(number)], capture_output=True, text=True)
        return result.stdout.strip().rsplit(":", 1)[1]

    process = None
    try:
        run(base + ["config", "--quiet"])
        run(base + ["up", "-d", "--wait", "--wait-timeout", "120", "database", "rabbitmq"])
        env["TEST_DATABASE_URL"] = f"postgres://amer:{password}@127.0.0.1:{port('database', 5432)}/amer_prototype?sslmode=disable"
        env["TEST_RABBITMQ_URL"] = f"amqp://amer:{broker_password}@127.0.0.1:{port('rabbitmq', 5672)}/"
        env["MIGRATION_DATABASE_URL"] = env["TEST_DATABASE_URL"]
        run(["make", "migrate.up", "MIGRATION_VERSION=13"])
        with tempfile.TemporaryDirectory(prefix="irp-control-") as directory:
            env["TEST_RABBITMQ_CONTROL_DIR"] = directory
            process = subprocess.Popen(["go", "test", "-race", "./test/integration/...", "-run",
                                        "^TestInterviewReportRabbitMQ", "-count=1", "-timeout=150s", "-v"],
                                       cwd=ROOT, env=env)
            deadline = time.monotonic() + 150
            stopped = False
            while process.poll() is None:
                if not stopped and (Path(directory) / "ready").exists():
                    run(base + ["stop", "--timeout", "5", "rabbitmq"])
                    stopped = True
                if time.monotonic() >= deadline:
                    raise RuntimeError("report broker verification exceeded its time budget")
                time.sleep(0.1)
            if process.returncode:
                raise SystemExit(process.returncode)
            if not stopped:
                raise RuntimeError("broker outage proof did not execute")
            print("PASS: real RabbitMQ/PostgreSQL confirm, redelivery, duplicate, outage and database-error proofs; no sqlmock", flush=True)
    finally:
        if process is not None and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
        run(base + ["down", "--volumes", "--remove-orphans"])


if __name__ == "__main__":
    main()
