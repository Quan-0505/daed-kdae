"""Copy shared state into a Go schema and wrap Rust DNS/routing section bodies."""
import argparse
import pathlib
import re
import sqlite3
import os
from contextlib import closing

TABLES = ('users', 'configs', 'dns', 'routings', 'subscriptions', 'nodes', 'groups',
          'group_nodes', 'group_subscriptions', 'group_policy_params', 'systems',
          'log_settings', 'node_latency_results')
LEGACY_TABLES = ('log_settings', 'node_latency_results')

def full_section(text, section):
    prefix = r'^\s*(?:\#[^\n]*\n\s*)*' + re.escape(section) + r'\s*\{'
    return text if re.match(prefix, text) else f'{section} {{\n{text}\n}}\n'

def migrate(source, template, output):
    if output.exists():
        raise ValueError('Output already exists; preserve it before migrating.')
    source = source.resolve(strict=True)
    template = template.resolve(strict=True)
    output.parent.mkdir(parents=True, exist_ok=True)
    with closing(sqlite3.connect(f'file:{source}?mode=ro', uri=True)) as src, \
         closing(sqlite3.connect(f'file:{template}?mode=ro', uri=True)) as schema, \
         closing(sqlite3.connect(output)) as dst:
        schema.backup(dst)
        os.chmod(output, 0o600)
        counts = {}
        for table in TABLES:
            available = {row[1] for row in src.execute(f'pragma table_info({table})')}
            if table in LEGACY_TABLES:
                if not available:
                    counts[table] = 0
                    continue
                if not dst.execute(f'pragma table_info({table})').fetchall():
                    # Fresh Go schemas do not create Rust log/cache tables.
                    # Preserve their data for recovery without treating them as
                    # active Go log settings or current latency measurements.
                    ddl = src.execute('select sql from sqlite_master where type=? and name=?',
                                      ('table', table)).fetchone()[0]
                    dst.execute(ddl)
            fields = [row[1] for row in dst.execute(f'pragma table_info({table})') if row[1] in available]
            if not fields:
                raise ValueError(f'Missing shared table: {table}')
            quoted = ','.join(f'"{field}"' for field in fields)
            rows = src.execute(f'select {quoted} from {table}').fetchall()
            dst.execute(f'delete from {table}')
            placeholders = ','.join('?' for _ in fields)
            dst.executemany(f'insert into {table}({quoted}) values({placeholders})', rows)
            counts[table] = len(rows)
        for table, column, section in [('dns', 'dns', 'dns'), ('routings', 'routing', 'routing')]:
            for id, text in dst.execute(f'select id,{column} from {table}').fetchall():
                dst.execute(f'update {table} set {column}=? where id=?', (full_section(text, section), id))
        dst.commit()
    return counts

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=pathlib.Path, help='Consistent Rust daed.db backup')
    parser.add_argument('template', type=pathlib.Path, help='Go-created wing.db schema')
    parser.add_argument('output', type=pathlib.Path, help='New independent wing.db')
    args = parser.parse_args()
    print(migrate(args.source, args.template, args.output))
