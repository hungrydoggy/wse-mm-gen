package main

import (
  "database/sql"
  "fmt"
  "log"
  "os"
  _ "github.com/go-sql-driver/mysql"
  godotenv "github.com/joho/godotenv"

  gen          "./gen"
  table_schema "./table_schema"
)

func main() {
  err := godotenv.Load()
  if err != nil {
    log.Fatal("Error loading .env file")
  }

  // connect db
  db, _ := sql.Open(
      "mysql",
      fmt.Sprintf("%s:%s@tcp(%s)/%s",
        os.Getenv("DB_USER"    ),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_HOST"    ),
        os.Getenv("DB_NAME"    ),
      ),
  )
  defer db.Close()


  // fetch table names
  table_names := table_schema.FetchTableNames(db)
  fmt.Println(table_names)


  // prepare out dir
  os.RemoveAll("./out")
  os.Mkdir("./out"            , 0755)
  os.Mkdir("./out/models"     , 0755)
  os.Mkdir("./out/view_models", 0755)


  /// generate crud codes
  // make schema_infos
  tablename_schemainfo_map := map[string]table_schema.SchemaInfo{}
  for _, tn := range table_names {
    schema := table_schema.FetchTableSchema(db, tn)
    
    tablename_schemainfo_map[tn] = table_schema.SchemaInfo{
      tn,
      schema,
      map[string]string{},
    }
  }

  // make schemainfo.Manyname_modelname_map
  for _, info := range tablename_schemainfo_map {
    for _, sch := range info.Schema {
      if sch.FieldType != table_schema.ASSOCIATION {
        continue
      }
      if sch.Association_info.Many_name != "" {
        target_model := tablename_schemainfo_map[sch.Association_info.Model_name]
        target_model.Manyname_modelname_map[sch.Association_info.Many_name] = info.Table_name
      }
    }
  }

  // generate Model for crud
  for _, info := range tablename_schemainfo_map {
    gen.GenModelForCrud(info.Table_name, info.Schema, info.Manyname_modelname_map)
  }

  // generate api
  gen.GenApi(tablename_schemainfo_map)
}
