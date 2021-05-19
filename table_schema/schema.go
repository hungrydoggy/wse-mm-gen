package table_schema

import (
  "database/sql"
  "fmt"
  "log"
  "strings"

  _ "github.com/go-sql-driver/mysql"
)


func FetchTableNames (db *sql.DB) []string {
  rows, err := db.Query("show tables")
  if err != nil {
    log.Fatal(err)
  }
  defer rows.Close()

  table_names := []string{}
  for rows.Next() {
    tn := ""
    rows.Scan(&tn)
    table_names = append(table_names, tn)
  }

  return table_names
}

func FetchTableSchema (db *sql.DB, table_name string) []*TableScheme {
  rows, err := db.Query(
      fmt.Sprintf("show full columns from %s", table_name),
  )
  if err != nil {
    log.Fatal(err)
  }
  defer rows.Close()

  schema := []*TableScheme{}
  for rows.Next() {
    var __ sql.NullString
    null_str := ""
    ts := TableScheme{}
    rows.Scan(
      &ts.Field,
      &ts.Type,
      &__,
      &null_str,
      &ts.Key,
      &ts.Default,
      &ts.Extra,
      &__,
      &ts.Comment,
    )
    ts.Null = true
    if null_str != "YES" {
      ts.Null = false
    }

    parseComment(&ts)

    schema = append(schema, &ts)
  }

  return schema
}

type FieldType int
const (
  NORMAL      FieldType = iota
  HIDDEN
  ASSOCIATION
)

type TableScheme struct {
  Field   string
  Type    string
  Null    bool
  Key     string
  Default sql.NullString
  Extra   string
  Comment string

  FieldType        FieldType
  Hidden_perms     []string
  Association_info AssociationInfo
}

type AssociationInfo struct {
  As_name    string
  Model_name string
  Model_fk   string
  Many_name  string
}



func parseComment (ts *TableScheme) {
  setting_str, comment := extractSettingWithComment(ts)
  if len(setting_str) <= 0 {
    return
  }
  
  switch setting_str[0] {
  case '#':
    ts.FieldType = HIDDEN
    parts := strings.Split(setting_str, ".")
    ts.Field = parts[0] // head
    ts.Hidden_perms = strings.Split(parts[1], ":") // tail
  case '@':
    ts.FieldType = ASSOCIATION
    parts := strings.Split(setting_str, ".")

    // head
    head := parts[0]
    head_elms := strings.Split(head, ":")
    ts.Field = head_elms[0]
    as_name := ts.Field[:len(ts.Field)-3]
    if len(head_elms) >= 2 {
      as_name = head_elms[1]
    }
    if as_name[0] == '@' {
      as_name = as_name[1:]
    }

    // middle
    middle := parts[1]
    middle_elms := strings.Split(middle, ":")
    model_name := middle_elms[0]
    model_fk   := "id"
    many_name  := ""
    if len(middle_elms) >= 2 {
      model_fk = middle_elms[1]
    }
    if len(middle_elms) >= 3 {
      many_name = middle_elms[2]
    }

    ts.Association_info = AssociationInfo{
      as_name,
      model_name,
      model_fk,
      many_name,
    }
  default:
    ts.FieldType = NORMAL
  }

  ts.Comment = comment
}

func extractSettingWithComment (ts *TableScheme) (string, string) {
  lines := strings.Split(ts.Comment, "\n")

  first_line := strings.Trim(lines[0], " ")
  if len(first_line) <= 0 || first_line[0] != '&' {
    return "", strings.Join(lines, "\n")
  }else {
    setting_str := first_line[1:]
    if setting_str[0] == ':' || setting_str[0] == '.' {
      setting_str = ts.Field + setting_str
    }else {
      setting_str = ts.Field + "." + setting_str
    }
    return setting_str, strings.Join(lines[1:], "\n")
  }
}


type SchemaInfo struct {
  Table_name             string
  Schema                 []*TableScheme
  Manyname_modelname_map map[string]string
}
