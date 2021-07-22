package table_schema

import (
  "database/sql"
  "fmt"
  "log"
  "regexp"
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

func FetchTableOptionsFromComment (db *sql.DB, table_name string) *TableOptions {
  tables, err := db.Query(
      fmt.Sprintf("show table status like '%s'", table_name),
  )
  if err != nil {
    log.Fatal(err)
  }
  tables.Next()
  var __ sql.NullString
  table_comment := ""
  tables.Scan(
    &__, &__, &__, &__, &__, &__, &__, &__, &__, &__,
    &__, &__, &__, &__, &__, &__, &__,
    &table_comment,
    &__, &__,
  )
  defer tables.Close()


  // parse table comment
  table_options := TableOptions{
  }
  for _, line := range strings.Split(table_comment, "\n") {
    if len(line) <= 0 || line[0] != '&' {
      continue
    }
    switch {
    case strings.HasPrefix(line, "&many_to_many("):
      sm := re_table_options_many_to_many.FindStringSubmatch(line)
      table_options.Many_to_many = []bool{}
      for _, v := range strings.Split(sm[1], ",") {
        if strings.Trim(v, " ") == "true" {
          table_options.Many_to_many = append(table_options.Many_to_many, true )
        }else {
          table_options.Many_to_many = append(table_options.Many_to_many, false)
        }
      }
    }
  }

  return &table_options
}

func FetchTableSchema (db *sql.DB, table_name string) []*TableScheme {
  rows, err := db.Query(
      fmt.Sprintf("show full columns from %s", table_name),
  )
  if err != nil {
    log.Fatal(err)
  }
  defer rows.Close()


  // parse comment of rows
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
  ASSO_HIDDEN
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


func parseComment_setHidden (ts *TableScheme, setting_str string) {
  ts.FieldType = HIDDEN
  parts := strings.Split(setting_str, ".")
  ts.Field = parts[0] // head
  ts.Hidden_perms = strings.Split(parts[1], ":") // tail
}

func parseComment_setAssociation (ts *TableScheme, setting_str string) {
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
}

func parseComment (ts *TableScheme) {
  setting_str, comment := extractSettingWithComment(ts.Field, ts.Comment)
  if len(setting_str) <= 0 {
    return
  }
  
  switch setting_str[0] {
  case '#':
    parseComment_setHidden(ts, setting_str)
  case '@':
    if setting_str[1] != '#' {
      // association
      parseComment_setAssociation(ts, setting_str)
    }else {
      // association-hidden
      hd_setting_str, cmt := extractSettingWithComment(ts.Field, comment)
      comment = cmt
      parseComment_setAssociation(ts, setting_str   )
      parseComment_setHidden     (ts, hd_setting_str)
      ts.FieldType = ASSO_HIDDEN
    }
  default:
    ts.FieldType = NORMAL
  }

  ts.Comment = comment
}

func extractSettingWithComment (field string, comment string) (string, string) {
  lines := strings.Split(comment, "\n")

  first_line := strings.Trim(lines[0], " ")
  if len(first_line) <= 0 || first_line[0] != '&' {
    return "", strings.Join(lines, "\n")
  }else {
    setting_str := first_line[1:]
    if setting_str[0] == ':' || setting_str[0] == '.' {
      setting_str = field + setting_str
    }else {
      setting_str = field + "." + setting_str
    }
    return setting_str, strings.Join(lines[1:], "\n")
  }
}


type SchemaInfo struct {
  Table_name             string
  Schema                 []*TableScheme
  Manyname_modelname_map map[string]string
  Through_names          []string
}

type TableOptions struct {
  Many_to_many []bool
}

var re_table_options_many_to_many = regexp.MustCompile("&many_to_many\\((.*)\\)")
