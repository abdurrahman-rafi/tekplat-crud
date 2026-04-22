package store

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]{0,63}$`)

// isValidColumnType allows both exact and parameterized MySQL types.
func isValidColumnType(t string) bool {
	exact := map[string]bool{
		"INT": true, "BIGINT": true, "FLOAT": true, "TEXT": true,
		"DATE": true, "DATETIME": true, "TIMESTAMP": true, "BOOLEAN": true, "JSON": true,
	}
	if exact[t] {
		return true
	}
	patterns := []string{
		`^VARCHAR\([1-9][0-9]{0,4}\)$`,
		`^CHAR\([1-9][0-9]{0,2}\)$`,
		`^DECIMAL\([1-9][0-9]?, ?[0-9]+\)$`,
	}
	for _, p := range patterns {
		if ok, _ := regexp.MatchString(p, t); ok {
			return true
		}
	}
	return false
}

type Column struct {
	Name       string
	Type       string
	References string // "table_name.column" or ""
}

type ColumnInfo struct {
	Name     string
	DataType string
	Key      string // "PRI", "MUL", ""
}

type FKInfo struct {
	Column    string
	RefTable  string
	RefColumn string
}

type TableSchema struct {
	Name    string
	Columns []ColumnInfo
	FKs     []FKInfo
}

type TableStore struct {
	db *sql.DB
}

func NewTableStore(db *sql.DB) *TableStore {
	return &TableStore{db: db}
}

// ── Table DDL ─────────────────────────────────────────────────────────────────

func (s *TableStore) ListTables(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func (s *TableStore) CreateTable(ctx context.Context, name string, columns []Column) error {
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("nama tabel tidak valid: %q", name)
	}
	if len(columns) == 0 {
		return fmt.Errorf("minimal satu kolom diperlukan")
	}

	colDefs := make([]string, 0, len(columns))
	var fkDefs []string

	for _, col := range columns {
		if !validIdentifier.MatchString(col.Name) {
			return fmt.Errorf("nama kolom tidak valid: %q", col.Name)
		}
		if !isValidColumnType(col.Type) {
			return fmt.Errorf("tipe kolom tidak valid: %q", col.Type)
		}
		colDefs = append(colDefs, fmt.Sprintf("`%s` %s", col.Name, col.Type))

		if col.References != "" {
			parts := strings.SplitN(col.References, ".", 2)
			if len(parts) == 2 && validIdentifier.MatchString(parts[0]) && validIdentifier.MatchString(parts[1]) {
				fkDefs = append(fkDefs, fmt.Sprintf(
					"FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`)",
					col.Name, parts[0], parts[1],
				))
			}
		}
	}

	parts := append([]string{"id INT NOT NULL AUTO_INCREMENT"}, colDefs...)
	parts = append(parts, "PRIMARY KEY (id)")
	parts = append(parts, fkDefs...)
	query := fmt.Sprintf("CREATE TABLE `%s` (%s)", name, strings.Join(parts, ", "))

	_, err := s.db.ExecContext(ctx, query)
	return err
}

func (s *TableStore) DropTable(ctx context.Context, name string) error {
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("nama tabel tidak valid: %q", name)
	}
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("DROP TABLE `%s`", name))
	return err
}

func (s *TableStore) AddForeignKey(ctx context.Context, table, col, refTable, refCol string) error {
	for _, v := range []string{table, col, refTable, refCol} {
		if !validIdentifier.MatchString(v) {
			return fmt.Errorf("identifier tidak valid: %q", v)
		}
	}
	constraintName := fmt.Sprintf("fk_%s_%s", table, col)
	query := fmt.Sprintf(
		"ALTER TABLE `%s` ADD CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`)",
		table, constraintName, col, refTable, refCol,
	)
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// ── Column info ───────────────────────────────────────────────────────────────

func (s *TableStore) ListColumns(ctx context.Context, tableName string) ([]string, error) {
	if !validIdentifier.MatchString(tableName) {
		return nil, fmt.Errorf("nama tabel tidak valid: %q", tableName)
	}
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("SHOW COLUMNS FROM `%s`", tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var field sql.NullString
		var rest [5]sql.NullString
		if err := rows.Scan(&field, &rest[0], &rest[1], &rest[2], &rest[3], &rest[4]); err != nil {
			return nil, err
		}
		columns = append(columns, field.String)
	}
	return columns, rows.Err()
}

// ── Row CRUD ──────────────────────────────────────────────────────────────────

func (s *TableStore) ListRows(ctx context.Context, tableName string) (cols []string, rows [][]string, err error) {
	if !validIdentifier.MatchString(tableName) {
		return nil, nil, fmt.Errorf("nama tabel tidak valid: %q", tableName)
	}
	r, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM `%s` LIMIT 200", tableName))
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	cols, err = r.Columns()
	if err != nil {
		return nil, nil, err
	}

	for r.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := r.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		row := make([]string, len(cols))
		for i, v := range raw {
			row[i] = anyToString(v)
		}
		rows = append(rows, row)
	}
	return cols, rows, r.Err()
}

func (s *TableStore) GetRowByID(ctx context.Context, tableName string, id int) (cols []string, vals []string, err error) {
	if !validIdentifier.MatchString(tableName) {
		return nil, nil, fmt.Errorf("nama tabel tidak valid: %q", tableName)
	}
	r, err := s.db.QueryContext(ctx, fmt.Sprintf("SELECT * FROM `%s` WHERE id = ?", tableName), id)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	cols, err = r.Columns()
	if err != nil {
		return nil, nil, err
	}
	if !r.Next() {
		return nil, nil, sql.ErrNoRows
	}

	raw := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range raw {
		ptrs[i] = &raw[i]
	}
	if err := r.Scan(ptrs...); err != nil {
		return nil, nil, err
	}
	vals = make([]string, len(cols))
	for i, v := range raw {
		vals[i] = anyToString(v)
	}
	return cols, vals, nil
}

func (s *TableStore) InsertRow(ctx context.Context, tableName string, cols, vals []string) error {
	if !validIdentifier.MatchString(tableName) {
		return fmt.Errorf("nama tabel tidak valid: %q", tableName)
	}
	escaped := make([]string, len(cols))
	placeholders := make([]string, len(cols))
	for i, c := range cols {
		if !validIdentifier.MatchString(c) {
			return fmt.Errorf("nama kolom tidak valid: %q", c)
		}
		escaped[i] = fmt.Sprintf("`%s`", c)
		placeholders[i] = "?"
	}
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName, strings.Join(escaped, ", "), strings.Join(placeholders, ", "))

	args := make([]any, len(vals))
	for i, v := range vals {
		args[i] = v
	}
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *TableStore) UpdateRow(ctx context.Context, tableName string, id int, cols, vals []string) error {
	if !validIdentifier.MatchString(tableName) {
		return fmt.Errorf("nama tabel tidak valid: %q", tableName)
	}
	sets := make([]string, len(cols))
	for i, c := range cols {
		if !validIdentifier.MatchString(c) {
			return fmt.Errorf("nama kolom tidak valid: %q", c)
		}
		sets[i] = fmt.Sprintf("`%s` = ?", c)
	}
	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE id = ?", tableName, strings.Join(sets, ", "))

	args := make([]any, len(vals)+1)
	for i, v := range vals {
		args[i] = v
	}
	args[len(vals)] = id
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *TableStore) DeleteRow(ctx context.Context, tableName string, id int) error {
	if !validIdentifier.MatchString(tableName) {
		return fmt.Errorf("nama tabel tidak valid: %q", tableName)
	}
	_, err := s.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM `%s` WHERE id = ?", tableName), id)
	return err
}

// ── Schema / ER Diagram ───────────────────────────────────────────────────────

func (s *TableStore) GetSchema(ctx context.Context) ([]TableSchema, error) {
	colRows, err := s.db.QueryContext(ctx, `
		SELECT TABLE_NAME, COLUMN_NAME, DATA_TYPE, COLUMN_KEY
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		ORDER BY TABLE_NAME, ORDINAL_POSITION`)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	schemaMap := make(map[string]*TableSchema)
	var order []string

	for colRows.Next() {
		var tbl, col, dtype, key string
		if err := colRows.Scan(&tbl, &col, &dtype, &key); err != nil {
			return nil, err
		}
		if _, ok := schemaMap[tbl]; !ok {
			schemaMap[tbl] = &TableSchema{Name: tbl}
			order = append(order, tbl)
		}
		schemaMap[tbl].Columns = append(schemaMap[tbl].Columns, ColumnInfo{
			Name: col, DataType: dtype, Key: key,
		})
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}

	fkRows, err := s.db.QueryContext(ctx, `
		SELECT TABLE_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = DATABASE() AND REFERENCED_TABLE_NAME IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer fkRows.Close()

	for fkRows.Next() {
		var tbl, col, refTbl, refCol string
		if err := fkRows.Scan(&tbl, &col, &refTbl, &refCol); err != nil {
			return nil, err
		}
		if s, ok := schemaMap[tbl]; ok {
			s.FKs = append(s.FKs, FKInfo{Column: col, RefTable: refTbl, RefColumn: refCol})
		}
	}
	if err := fkRows.Err(); err != nil {
		return nil, err
	}

	result := make([]TableSchema, 0, len(order))
	for _, name := range order {
		result = append(result, *schemaMap[name])
	}
	return result, nil
}

// ── CSV import ────────────────────────────────────────────────────────────────

func (s *TableStore) ImportCSV(ctx context.Context, tableName string, records [][]string) (int, error) {
	if !validIdentifier.MatchString(tableName) {
		return 0, fmt.Errorf("nama tabel tidak valid: %q", tableName)
	}
	if len(records) < 2 {
		return 0, fmt.Errorf("CSV minimal harus memiliki baris header dan satu baris data")
	}

	headers := records[0]
	for _, h := range headers {
		if !validIdentifier.MatchString(h) {
			return 0, fmt.Errorf("nama kolom CSV tidak valid: %q", h)
		}
	}

	escaped := make([]string, len(headers))
	placeholders := make([]string, len(headers))
	for i, h := range headers {
		escaped[i] = fmt.Sprintf("`%s`", h)
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		tableName, strings.Join(escaped, ", "), strings.Join(placeholders, ", "))

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	count := 0
	for _, row := range records[1:] {
		if len(row) != len(headers) {
			continue
		}
		args := make([]any, len(row))
		for i, v := range row {
			args[i] = v
		}
		if _, err := stmt.ExecContext(ctx, args...); err != nil {
			return 0, err
		}
		count++
	}
	return count, tx.Commit()
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func anyToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}
