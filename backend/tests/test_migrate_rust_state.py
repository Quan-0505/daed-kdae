import importlib.util
import pathlib
import tempfile
import sqlite3
import unittest
from contextlib import closing

spec = importlib.util.spec_from_file_location('migration', pathlib.Path(__file__).parents[1] / 'migrate-rust-state.py')
migration = importlib.util.module_from_spec(spec)
spec.loader.exec_module(migration)

class MigrationTest(unittest.TestCase):
    def test_nested_dns_body_gets_outer_section(self):
        body = 'upstream { local: "udp://127.0.0.1:53" }\nrouting { request { fallback: local } }'
        self.assertEqual(migration.full_section(body, 'dns'), 'dns {\n' + body + '\n}\n')

    def test_existing_section_with_comment_is_preserved(self):
        text = '# existing config\nrouting { fallback: direct }\n'
        self.assertEqual(migration.full_section(text, 'routing'), text)

    def test_shared_values_are_copied_into_go_schema_without_rust_columns(self):
        with tempfile.TemporaryDirectory() as folder:
            folder = pathlib.Path(folder)
            source, schema, output = [folder / name for name in ('rust.db', 'go.db', 'out.db')]
            for path, extra in [(source, ', rust_only text'), (schema, '')]:
                with closing(sqlite3.connect(path)) as db, db:
                    for table in migration.TABLES:
                        column = 'dns' if table == 'dns' else 'routing' if table == 'routings' else 'value'
                        db.execute(f'create table {table}(id integer primary key,{column} text{extra})')
                        db.execute(f'insert into {table}(id,{column}) values(7,?)', ('fallback: direct',))
            migration.migrate(source, schema, output)
            with closing(sqlite3.connect(output)) as db, db:
                self.assertEqual(db.execute('select id,routing from routings').fetchone(), (7, 'routing {\nfallback: direct\n}\n'))
                self.assertEqual(db.execute('select value from users').fetchone()[0], 'fallback: direct')
            with closing(sqlite3.connect(source)) as db, db:
                self.assertEqual(db.execute('select routing from routings').fetchone()[0], 'fallback: direct')
            with self.assertRaises(ValueError):
                migration.migrate(source, schema, output)

    def test_fresh_go_schema_preserves_optional_rust_cache_tables(self):
        with tempfile.TemporaryDirectory() as folder:
            folder = pathlib.Path(folder)
            source, schema, output = [folder / name for name in ('rust.db', 'go.db', 'out.db')]
            for path in (source, schema):
                with closing(sqlite3.connect(path)) as db, db:
                    for table in migration.TABLES:
                        if path == schema and table in migration.LEGACY_TABLES:
                            continue
                        column = 'dns' if table == 'dns' else 'routing' if table == 'routings' else 'value'
                        db.execute(f'create table {table}(id integer primary key,{column} text)')
                        db.execute(f'insert into {table}(id,{column}) values(7,?)', ('fallback: direct',))
            counts = migration.migrate(source, schema, output)
            with closing(sqlite3.connect(output)) as db, db:
                for table in migration.LEGACY_TABLES:
                    self.assertEqual(counts[table], 1)
                    self.assertEqual(db.execute(f'select value from {table}').fetchone()[0], 'fallback: direct')

if __name__ == '__main__':
    unittest.main()
