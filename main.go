package main

import (
  "database/sql"
  "fmt"
  "log"
  "os"
  "sort"
  "strings"
  _ "github.com/go-sql-driver/mysql"
  godotenv "github.com/joho/godotenv"
  funk     "github.com/thoas/go-funk"

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
  os.Mkdir("./out"               , 0755)
  os.Mkdir("./out/models"        , 0755)
  os.Mkdir("./out/view_models"   , 0755)
  os.Mkdir("./out/custom_results", 0755)


  /// generate crud codes
  // make schema_infos
  tablename_schemainfo_map := map[string]*table_schema.SchemaInfo{}
  for _, tn := range table_names {
    schema := table_schema.FetchTableSchema(db, tn)
    
    tablename_schemainfo_map[tn] = &table_schema.SchemaInfo{
      tn,
      schema,
      map[string]string{},
      []string{},
    }
  }

  // make schemainfo.Manyname_modelname_map
  for _, info := range tablename_schemainfo_map {
    for _, sch := range info.Schema {
      if sch.FieldType != table_schema.ASSOCIATION && sch.FieldType != table_schema.ASSO_HIDDEN {
        continue
      }
      if sch.Association_info.Many_name != "" {
        target_model := tablename_schemainfo_map[sch.Association_info.Model_name]
        target_model.Manyname_modelname_map[sch.Association_info.Many_name] = info.Table_name
      }
    }
  }

  // check table options
  for tn, info := range tablename_schemainfo_map {
    options := table_schema.FetchTableOptionsFromComment(db, tn)

    if options.Many_to_many != nil {
      // add many-to-many to schemainfo.Manyname_modelname_map
      assocation_schemes := funk.Filter(
          info.Schema,
          func (sch *table_schema.TableScheme) bool {
            return sch.FieldType == table_schema.ASSOCIATION || sch.FieldType == table_schema.ASSO_HIDDEN
          },
      ).([]*table_schema.TableScheme)
      if len(options.Many_to_many) > 0 && options.Many_to_many[0] {
        model_info := tablename_schemainfo_map[assocation_schemes[0].Association_info.Model_name]
        many_name := assocation_schemes[1].Field[1:]
        many_name = gen.MakePlural(many_name[:len(many_name)-3])
        model_info.Manyname_modelname_map[many_name] = assocation_schemes[1].Association_info.Model_name

        // add through name
        target_info := tablename_schemainfo_map[assocation_schemes[1].Association_info.Model_name]
        target_info.Through_names = append(target_info.Through_names, info.Table_name)
      }
      if len(options.Many_to_many) > 1 && options.Many_to_many[1] {
        model_info := tablename_schemainfo_map[assocation_schemes[1].Association_info.Model_name]
        many_name := assocation_schemes[1].Field[1:]
        many_name = gen.MakePlural(many_name[:len(many_name)-3])
        model_info.Manyname_modelname_map[many_name] = assocation_schemes[0].Association_info.Model_name

        // add through name
        target_info := tablename_schemainfo_map[assocation_schemes[0].Association_info.Model_name]
        target_info.Through_names = append(target_info.Through_names, info.Table_name)
      }
    }
  }

  // sort schema
  for _, info := range tablename_schemainfo_map {
    sort.Slice(
        info.Schema,
        func (a, b int) bool {
          return strings.Compare(info.Schema[a].Field, info.Schema[b].Field) < 0;
        },
    )
  }

  // generate Model for crud
  for _, info := range tablename_schemainfo_map {
    sort.Strings(info.Through_names)
    gen.GenModelForCrud(info.Table_name, info.Schema, info.Manyname_modelname_map, info.Through_names)
  }

  // generate api
  gen.GenApi(tablename_schemainfo_map)
}
