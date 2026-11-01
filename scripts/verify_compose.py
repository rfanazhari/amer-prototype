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
env["COMPOSE_PROJECT_NAME"] = project
base = ["docker", "compose", "--project-name", project, "--profile", "redis", "--profile", "rabbitmq"]
stack = "--stack" in sys.argv


def run(args, **kwargs):
    """Execute a checked command with isolated project settings and inherited credentials."""
    return subprocess.run(args, cwd=root, env=env, check=True, **kwargs)


def port(service, container_port):
    """Read the ephemeral host port assigned to one service in this verification project."""
    result = run(base + ["port", service, str(container_port)], capture_output=True, text=True)
    return result.stdout.strip().rsplit(":", 1)[1]


def sql(database, statement):
    """Execute fixture SQL in this disposable project without printing rows or credentials."""
    return run(base + ["exec", "-T", "database", "psql", "-X", "-v", "ON_ERROR_STOP=1",
                       "-U", "amer", "-d", database, "-At"],
               input=statement, capture_output=True, text=True).stdout.strip()


def database_url(database):
    """Build an environment-only URL for an isolated verification database."""
    return f"postgres://amer:{secret}@127.0.0.1:{port('database', 5432)}/{database}?sslmode=disable"


def snapshot(database):
    """Capture complete retained rows for equality without printing contacts or hashes."""
    return sql(database, """
        SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM tasks t;
        SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM administrative_accounts t;
        SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM administrative_sessions t;
    """)


def container_smoke():
    """Check the packaged binary on the retained schema without printing credentials."""
    url = "http://127.0.0.1:" + port("app", 8080)
    deadline = time.monotonic() + 10
    while True:
        try:
            with urllib.request.urlopen(url + "/health", timeout=2) as response:
                assert json.load(response) == {"status": "success", "data": {}}
            break
        except OSError:
            if time.monotonic() >= deadline:
                raise
            time.sleep(0.1)
    request = urllib.request.Request(url + "/tasks", data=b'{"title":"rollback smoke"}',
                                     headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(request, timeout=2) as response:
        task = json.load(response)["data"]["id"]
    for operation in ("start", "complete"):
        request = urllib.request.Request(url + "/tasks/" + task + "/" + operation, data=b'{}',
                                         headers={"Content-Type": "application/json"})
        with urllib.request.urlopen(request, timeout=2) as response:
            assert response.status == 200
    request = urllib.request.Request(url + "/admin/sessions", data=json.dumps({
        "email": env["ACCESS_BOOTSTRAP_ADMIN_EMAIL"], "password": env["ACCESS_BOOTSTRAP_ADMIN_PASSWORD"]
    }).encode(), headers={"Content-Type": "application/json", "Origin": env["ACCESS_HTTP_ORIGIN"]})
    with urllib.request.urlopen(request, timeout=3) as response:
        assert response.status == 201


try:
    run(base + ["config", "--quiet"])
    run(base + ["up", "-d", "--wait", "--wait-timeout", "90", "database"])
    env["TEST_DATABASE_URL"] = database_url("amer_prototype")
    env["MIGRATION_DATABASE_URL"] = env["TEST_DATABASE_URL"]
    # Prepare explicitly before either container or in-process application startup.
    run(["make", "migrate.up", "MIGRATION_VERSION=9"])
    run(["make", "migrate.status"])
    sql("amer_prototype", "CREATE DATABASE migration_proof")
    env["TEST_MIGRATION_DATABASE_URL"] = database_url("migration_proof")
    run(["go", "test", "-race", "./infrastructure/database/schema/...", "-count=1", "-v"])
    # Exercise the public Make surface for a disposable reversal as well as the adapter.
    sql("migration_proof", "DROP SCHEMA public CASCADE; CREATE SCHEMA public")
    env["MIGRATION_DATABASE_URL"] = env["TEST_MIGRATION_DATABASE_URL"]
    run(["make", "migrate.up", "MIGRATION_VERSION=2"])
    run(["make", "migrate.down", "MIGRATION_STEPS=2", "MIGRATION_DISPOSABLE=yes"])
    run(["make", "migrate.up", "MIGRATION_VERSION=2"])
    run(["make", "migrate.status"])
    env["MIGRATION_DATABASE_URL"] = env["TEST_DATABASE_URL"]
    if stack:
        run(base + ["build", "app"])
        env["MIGRATION_VERSION"] = "9"
        # Exercise the packaged command and immutable SQL as a separate pre-start job.
        run(base + ["run", "--rm", "--no-deps", "migrate", "up"])
        run(["make", "docker.up"])
        run(base + ["up", "-d", "--wait", "--wait-timeout", "120"])
        container_smoke()
        fallback_image = run(["docker", "image", "inspect", env["APP_IMAGE"], "--format", "{{.Id}}"], capture_output=True, text=True).stdout.strip()
        print("Container HTTP health and optional service readiness passed", flush=True)
        print("Recorded compatible fallback image: " + fallback_image, flush=True)
        # Stop all app writers before backup/cutover rehearsal and destructive test setup.
        run(base + ["stop", "app"])
    print("Running real PostgreSQL/API lifecycle checks against isolated Compose database", flush=True)
    run(["go", "test", "-race", "-p", "1", "./test/smoke/...", "./test/integration/...", "-count=1", "-v"])
    before = snapshot("amer_prototype")
    backup = run(base + ["exec", "-T", "database", "pg_dump", "-U", "amer", "-d", "amer_prototype", "-Fc"], capture_output=True).stdout
    sql("amer_prototype", "CREATE DATABASE cutover_restore")
    run(base + ["exec", "-T", "database", "pg_restore", "-U", "amer", "-d", "cutover_restore", "--exit-on-error"], input=backup, capture_output=True)
    assert before == snapshot("cutover_restore"), "restored data differs"
    # This clone represents a populated full legacy schema with no migration metadata.
    sql("cutover_restore", "DROP TABLE schema_migrations")
    env["MIGRATION_DATABASE_URL"] = database_url("cutover_restore")
    run(["make", "migrate.baseline"])
    run(["make", "migrate.up", "MIGRATION_VERSION=9"])
    run(["make", "migrate.status"])
    assert before == snapshot("cutover_restore"), "adoption changed retained data"
    env["TEST_DATABASE_URL"] = env["MIGRATION_DATABASE_URL"]
    run(["go", "test", "-race", "-p", "1", "./test/smoke/...", "./test/integration/...",
         "-run", "TestTaskAPIFlow|TestPersistenceAcrossRuntimeRestart|TestAdministrativeHTTP", "-count=1", "-v"])
    if stack:
        retained = snapshot("amer_prototype")
        # Roll back to the exact migration-aware image previously verified in this rehearsal.
        env["APP_IMAGE"] = fallback_image
        run(base + ["up", "-d", "--no-build", "--force-recreate", "app"])
        assert retained == snapshot("amer_prototype"), "rollback changed retained rows"
        container_smoke()
        run(base + ["stop", "app"])
        print("Rollback to recorded compatible image passed Task/admin HTTP smoke on retained version 9", flush=True)
    print("Populated backup/restore, guarded adoption, compatible restart and Task/admin smoke passed; row equality verified without secret output", flush=True)
finally:
    subprocess.run(base + ["down", "--volumes"], cwd=root, env=env, check=True)
