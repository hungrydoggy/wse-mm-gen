package gen

import (
  "fmt"
  "io/ioutil"
  "os"
  "regexp"
  "sort"
  "strings"

  funk "github.com/thoas/go-funk"

  table_schema "../table_schema"
)

func GenApi (tablename_schemainfo_map map[string]table_schema.SchemaInfo) {
  // create api source code
  f, err := os.Create("./out/api.dart")
  check(err)
  defer f.Close()


  // import part
  _, err = f.WriteString(
      strings.Join(
        []string{
          "import 'package:mm/model.dart';",
          "import 'package:mm/property.dart';",
          "import 'package:wse_mm/wse_model.dart';",
        },
        "\n",
      ),
  )
  check(err)
  _, err = f.WriteString("\n\n")
  check(err)


  // prepare model_names
  model_names := []string{}
  for table_name := range tablename_schemainfo_map {
    model_names = append(model_names, table_name)
  }
  sort.Slice(
      model_names,
      func (a, b int) bool {
        return strings.Compare(model_names[a], model_names[b]) < 0;
      },
  )


  // write import-model part
  for _, mn := range model_names {
    _, err = f.WriteString(
        fmt.Sprintf("import './models/%s.dart';\n", mn),
    )
    check(err)
  }
  _, err = f.WriteString("\n")
  check(err)


  // write import-model part
  for _, mn := range model_names {
    _, err = f.WriteString(
        fmt.Sprintf("import './view_models/%sVM.dart';\n", mn),
    )
    check(err)
  }
  _, err = f.WriteString("\n\n\n")
  check(err)


  // write class head
  _, err = f.WriteString(api_head_str)
  check(err)


  // read api doc
  bytes, err := ioutil.ReadFile("./apis.md")
  if err != nil {
    panic(err)
  }
  doc := string(bytes)


  // generate
  idx := strings.Index(doc, "각 Api 설명")
  api_detail_part := doc[idx:]
  for _, idxes := range re_api_head.FindAllStringIndex(api_detail_part, -1) {
    start_idx := idxes[0]
    sm := re_api_head.FindStringSubmatch(api_detail_part[start_idx:])
    path       := sm[1]
    method     := sm[2]
    permission := sm[3]
    comment    := sm[4]

    permission = strings.Trim(permission, "_")

    fmt.Println("##", path, method)
    fmt.Println(permission)
    fmt.Println(comment)
    crud_sm := re_crud_api.FindStringSubmatch(comment)
    if len(crud_sm) > 0 {
      // crud api
      crud_type  := crud_sm[1]
      model_name := crud_sm[2]
      info, ok := tablename_schemainfo_map[model_name]
      if ok == false {
        panic("no model name : " + model_name)
      }
      genCrudApi(f, info, crud_type, path)

    }else {
      // TODO custom api
    }

    println("")
  }


  // tail
  _, err = f.WriteString("}\n")
  check(err)


  // end
  f.Sync()
}


func genCrudApi (f *os.File, info table_schema.SchemaInfo, crud_type string, path string) {
  switch crud_type {
  case "create":
    genCrudApi_create(f, info, path)
  case "read":
    if strings.Contains(path, "&lt;id&gt;") {
      // TODO get by id
    }else {
      genCrudApi_get(f, info, path)
    }
  case "update":
    // TODO update
  case "delete":
    // TODO delete
  default:
    panic("unknown crud_type " + crud_type)
  }
}

func genCrudApi_get (f *os.File, info table_schema.SchemaInfo, path string) {
  // head
  _, err := f.WriteString(
      fmt.Sprintf(
        "  Future<List<%[1]sVM>> get_%[2]s ({\n      required dynamic options,\n      dynamic? order_query,\n  }) async {",
        info.Table_name,
        makeFuncNameFromPath(path),
      ),
  )
  check(err)


  // write codes
  _, err = f.WriteString(
      fmt.Sprintf(api_crud_get_codes_fmt, info.Table_name),
  )
  check(err)


  // tail
  _, err = f.WriteString("  }\n\n")
  check(err)
}

func genCrudApi_create (f *os.File, info table_schema.SchemaInfo, path string) {
  // head
  _, err := f.WriteString(
      fmt.Sprintf(
        "  Future<%[1]sVM> post_%[2]s ({%[3]s\n  }) async {\n",
        info.Table_name,
        makeFuncNameFromPath(path),
        strings.Join(
          funk.Map(
            funk.Filter(
              info.Schema,
              func (sch table_schema.TableScheme) bool {
                return sch.Field != "id" && sch.Field != "createdAt" && sch.Field != "updatedAt";
              },
            ),
            func (sch table_schema.TableScheme) string {
              code := "\n      "
              if sch.Null == false && sch.Default.Valid == false {
                code += "required "
              }
              prop_type := convertTypeFromSql(sch.Type)
              if sch.Null == true {
                code += fmt.Sprintf(
                    "%s? %s",
                    prop_type,
                    makePropName(sch.Field),
                )
              }else {
                code += fmt.Sprintf(
                    "%s %s",
                    prop_type,
                    makePropName(sch.Field),
                )
              }
              if sch.Default.Valid == true {
                if prop_type == "String" {
                  code += fmt.Sprintf(` = "%s",`, sch.Default.String)
                }else if prop_type == "bool" {
                  if sch.Default.String == "0" {
                    code += " = false,"
                  }else {
                    code += " = true,"
                  }
                }else {
                  code += fmt.Sprintf(` = %s,`, sch.Default.String)
                }
              }else {
                code += ","
              }
              return code
            },
          ).([]string),
          "",
        ),
      ),
  )
  check(err)


  /// codes
  // write property_value_map - required
  _, err = f.WriteString("    var property_value_map = <Property, dynamic>{\n")
  check(err)
  required_schema := funk.Filter(
    info.Schema,
    func (sch table_schema.TableScheme) bool {
      return sch.Field != "id" && sch.Field != "createdAt" && sch.Field != "updatedAt" && sch.Null == false
    },
  ).([]table_schema.TableScheme)
  prop_max_len := int(funk.Reduce(
    required_schema,
    func (acc int, sch table_schema.TableScheme) int {
      return funk.MaxInt([]int{acc, len(makePropName(sch.Field))}).(int)
    },
    0,
  ))
  for _, sch := range required_schema {
    _, err = f.WriteString(
        fmt.Sprintf(
          "      %[1]s.em.%-[2]*[3]s: %[3]s,\n",
          info.Table_name,
          prop_max_len,
          makePropName(sch.Field),
        ),
    )
    check(err)
  }
  _, err = f.WriteString("    };\n")
  check(err)

  // write property_value_map - optional
  for _, sch := range info.Schema {
    if sch.Field == "id" || sch.Field == "createdAt" || sch.Field == "updatedAt" {
      continue
    }
    if sch.Null == true {
      _, err = f.WriteString(
          fmt.Sprintf(
            "    if (%[2]s != null)\n      property_value_map[%[1]s.em.%[2]s] = %[2]s;\n",
            info.Table_name,
            makePropName(sch.Field),
          ),
      )
      check(err)
    }
  }
  _, err = f.WriteString("\n")
  check(err)


  // create model
  _, err = f.WriteString(
      fmt.Sprintf(
        "    final m = (await Model.createModel(\n        %[1]s.mh,\n        property_value_map,\n    )) as %[1]s;\n",
        info.Table_name,
      ),
  )
  check(err)


  // return view model
  _, err = f.WriteString(
      fmt.Sprintf(
        "    return %sVM({\n",
        info.Table_name,
      ),
  )
  check(err)

  field_max_len := int(funk.Reduce(
    info.Schema,
    func (acc int, sch table_schema.TableScheme) int {
      return funk.MaxInt([]int{acc, len(sch.Field)}).(int)
    },
    0,
  ))

  for _, sch := range info.Schema {
    _, err = f.WriteString(
        fmt.Sprintf(
          "      %-[1]*[2]s: m.%[3]s,\n",
          field_max_len + 2,
          "'" + sch.Field + "'",
          makePropName(sch.Field),
        ),
    )
    check(err)
  }
  

  // tail
  _, err = f.WriteString("    });\n  }\n\n")
  check(err)
}

func makeFuncNameFromPath (path string) string {
  return strings.Join(
      funk.Map(
        strings.Split(path, "/"),
        func (p string) string {
          if p == "" {
            return ""
          }

          r := strings.Join(
            funk.Map(
              strings.Split(p, "-"),
              func (s string) string {
                return strings.ToUpper(s[0:1]) + s[1:]
              },
            ).([]string),
            "",
          )
          return strings.ToLower(r[0:1]) + r[1:]
        },
      ).([]string),
      "_",
  )
}



var re_api_head     = regexp.MustCompile("\n\\#\\# (.*)&nbsp;&nbsp;&nbsp;&nbsp;`(.*)`\n> permission: (.*)\n>.*\n> (.*)\n")
var re_api_request  = regexp.MustCompile(`\n\#\#\# Request` )
var re_api_response = regexp.MustCompile(`\n\#\#\# Response`)
var re_crud_api     = regexp.MustCompile("CRUD api - `(.*)` of (.*)")


var api_head_str = `
class Api {

`

var api_crud_get_codes_fmt = `
    final res_jsons = await WseModel.find(
        %[1]s.mh,
        options,
        order_query: order_query,
    );
    return res_jsons.map((e) => %[1]sVM(e)).toList();
`
