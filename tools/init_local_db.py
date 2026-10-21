"""Create project-only local databases; never change an existing role or database."""
import getpass
import os
from pathlib import Path
import secrets
import shutil
import subprocess

ROOT = Path(__file__).resolve().parent.parent


def main():
    env_file = ROOT / '.env'
    if env_file.exists():
        raise SystemExit('.env already exists; existing configuration was left unchanged.')
    psql = shutil.which('psql')
    if not psql and os.name == 'nt':
        candidate = Path('C:/Program Files/PostgreSQL/18/bin/psql.exe')
        if candidate.exists():
            psql = str(candidate)
    if not psql:
        raise SystemExit('psql is required (PostgreSQL 18 client).')
    env = os.environ.copy()
    env['PGPASSWORD'] = env.pop('KOC_PG_ADMIN_PASSWORD', None) or getpass.getpass('Local postgres password: ')
    command = [psql, '-h', '127.0.0.1', '-p', '5432', '-U', 'postgres', '-d', 'postgres', '-w', '-X', '-A', '-t', '-v', 'ON_ERROR_STOP=1']

    def sql(statement):
        result = subprocess.run(command, input=statement, text=True, capture_output=True, env=env)
        if result.returncode:
            raise SystemExit('Database setup failed. Check local PostgreSQL access. No existing data was removed.')
        return result.stdout.strip()

    existing = sql("SELECT rolname FROM pg_roles WHERE rolname = 'kocbenefits_local'; SELECT datname FROM pg_database WHERE datname IN ('kocbenefits_dev', 'kocbenefits_test');")
    if existing:
        raise SystemExit('Project role/database already exists. Configure .env manually; no existing credentials were changed.')
    password = secrets.token_hex(24)
    sql(f"CREATE ROLE kocbenefits_local LOGIN PASSWORD '{password}' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION;")
    sql('CREATE DATABASE kocbenefits_dev OWNER kocbenefits_local;')
    sql('CREATE DATABASE kocbenefits_test OWNER kocbenefits_local;')
    with env_file.open('x', encoding='utf-8') as f:
        f.write('APP_ENV=development\nHTTP_ADDR=127.0.0.1:8080\n')
        for key, db in [('DATABASE_URL', 'kocbenefits_dev'), ('TEST_DATABASE_URL', 'kocbenefits_test')]:
            f.write(f'{key}=postgres://kocbenefits_local:{password}@127.0.0.1:5432/{db}?sslmode=disable\n')
        f.write('SCHOOL_EMAIL_DOMAIN=stu.koc.k12.tr\nELIGIBILITY_POLICY=manual\nREGISTRATION_OPEN=true\n')
        f.write('APP_ORIGIN=http://127.0.0.1:5173\nLOCAL_EMAIL=true\n')
        f.write('SMTP_HOST=127.0.0.1\nSMTP_PORT=1025\nSMTP_FROM=noreply@kocbenefits.example\nSMTP_USERNAME=\nSMTP_PASSWORD=\n')
    print('Created kocbenefits_dev and kocbenefits_test with a dedicated local role. Credentials saved only in ignored .env.')


if __name__ == '__main__':
    main()
