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

func GenApi (tablename_schemainfo_map map[string]*table_schema.SchemaInfo) {
  // create api source code
  f, err := os.Create("./out/api.dart")
  check(err)
  defer f.Close()


  // read api doc
  bytes, err := ioutil.ReadFile("./apis.md")
  if err != nil {
    panic(err)
  }
  doc := string(bytes)


  // make ApiDocInfo list
  api_doc_infos := []ApiDocInfo{}
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

    api_doc_infos = append(
        api_doc_infos,
        ApiDocInfo{
          path,
          method,
          permission,
          comment,
          start_idx,
        },
    )
    println("")
  }


  // import part
  _, err = f.WriteString(
      strings.Join(
        []string{
          "import 'dart:convert';",
          "import 'dart:math';\n",
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


  // write import-view-model part
  for _, mn := range model_names {
    _, err = f.WriteString(
        fmt.Sprintf("import './view_models/%sVM.dart';\n", mn),
    )
    check(err)
  }
  _, err = f.WriteString("\n")
  check(err)


  // write import-custom-result part
  for _, ad_info := range api_doc_infos {
    crud_sm := re_crud_api.FindStringSubmatch(ad_info.Comment)
    if len(crud_sm) > 0 {
      continue
    }
    _, err = f.WriteString(
        fmt.Sprintf(
            "import './custom_results/%s_%s.dart';\n",
            makeModelNameFromPath(ad_info.Method),
            makeModelNameFromPath(ad_info.Path),
        ),
    )
    check(err)
  }
  _, err = f.WriteString("\n\n\n")
  check(err)


  // write class head
  _, err = f.WriteString(api_head_str)
  check(err)


  // generate
  for _, ad_info := range api_doc_infos {
    // extract request
    fmt.Println("!!", ad_info.Path, ad_info.Method)
    custom_api_part := api_detail_part[ad_info.Start_idx:]
    req_idxes := re_api_request.FindStringIndex(custom_api_part)
    res_idxes := re_api_response.FindStringIndex(custom_api_part[req_idxes[1]:])
    request := re_md_code.FindStringSubmatch(custom_api_part[req_idxes[1]:req_idxes[1]+res_idxes[0]])[1]
    request_jsonex := ParseJsonEx(request)

    // extract response
    other_idxes := re_other_statuses.FindStringIndex(custom_api_part[req_idxes[1]+res_idxes[1]:])
    response_part := custom_api_part[req_idxes[1]+res_idxes[1]: req_idxes[1]+res_idxes[1]+other_idxes[1]]
    sm_idxes_list := re_small_title.FindAllStringIndex(response_part, -1)
    response := re_md_code.FindStringSubmatch(response_part[sm_idxes_list[0][1]:sm_idxes_list[1][0]])[1]
    response_jsonex := ParseJsonEx(response)

    crud_sm := re_crud_api.FindStringSubmatch(ad_info.Comment)
    if len(crud_sm) > 0 {
      /// crud api
      crud_type  := crud_sm[1]
      model_name := crud_sm[2]
      info, ok := tablename_schemainfo_map[model_name]
      if ok == false {
        panic("no model name : " + model_name)
      }
      genCrudApi(f, info, crud_type, ad_info.Path, &ad_info, &request_jsonex, &response_jsonex)

    }else {
      /// custom api
      genCustomApi(f, tablename_schemainfo_map, &ad_info, &request_jsonex, &response_jsonex)
      GenCustomResult(tablename_schemainfo_map, &ad_info, &response_jsonex)
    }
  }


  // tail
  _, err = f.WriteString("}\n")
  check(err)


  // end
  f.Sync()
}


func genCustomApi (
    f *os.File,
    tablename_schemainfo_map map[string]*table_schema.SchemaInfo,
    ad_info *ApiDocInfo,
    request_jsonex  *JsonExValue,
    response_jsonex *JsonExValue,
) {
  result_class := fmt.Sprintf(
      "%s_%s",
      makeModelNameFromPath(ad_info.Method),
      makeModelNameFromPath(ad_info.Path),
  )


  /// head
  _, err := f.WriteString(
      fmt.Sprintf(
        "  static Future<%s> %s_%s (",
        result_class,
        makeFuncNameFromPath(ad_info.Method),
        makeFuncNameFromPath(ad_info.Path),
      ),
  )
  check(err)


  // req_params
  req_params := funk.Filter(
    funk.Map(
      strings.Split(ad_info.Path, "/"),
      func (p string) []string {
        return convertReqParamPair(p)
      },
    ),
    func (pair []string) bool {
      return pair != nil
    },
  ).([][]string)
  for _, pair := range req_params {
    _, err := f.WriteString(
        fmt.Sprintf(
          "\n    String %s,",
          pair[0],
        ),
    )
    check(err)
  }

  // params
  _, err = f.WriteString(
      "\n    {\n      // null : use WseModel.token\n      // other: use named-token\n      String? token_name,\n",
  )
  check(err)

  request_map := request_jsonex.Value.(map[string]JsonExValue)

  request_keys := []string{}
  for k := range request_map {
    request_keys = append(request_keys, k)
  }
  sort.Strings(request_keys)

  for _, k := range request_keys {
    v := request_map[k]
    is_optional := funk.Reduce(
        v.Comments,
        func (acc bool, c string) bool {
          return acc || strings.HasPrefix(strings.ToLower(strings.Trim(c, " ")), "optional")
        },
        false,
    ).(bool)
    optional_chr := ""
    required_str := "required "
    if is_optional {
      optional_chr = "?"
      required_str = ""
    }

    _, err := f.WriteString("\n")
    check(err)

    // comment
    for _, cmt := range v.Comments {
      _, err := f.WriteString(fmt.Sprintf("      //%s\n", cmt))
      check(err)
    }


    // param
    switch v.Type {
    case "object":
      _, err := f.WriteString(
          fmt.Sprintf("      %sdynamic%s %s,\n", required_str, optional_chr, makePropName(k)),
      )
      check(err)
    case "array":
      _, err := f.WriteString(
          fmt.Sprintf("      %sList<dynamic>%s %s,\n", required_str, optional_chr, makePropName(k)),
      )
      check(err)
    case "string":
      _, err := f.WriteString(
          fmt.Sprintf(
            "      %s%s%s %s,\n",
            required_str,
            convertTypeFromDoc(v.Value.(string)),
            optional_chr,
            makePropName(k),
          ),
      )
      check(err)
    default:
      panic("unknown json-ex type : " + v.Type)
    }
  }

    
  // end of api head
  _, err = f.WriteString("\n    }\n  ) async {\n")
  check(err)



  /// gen codes
  // params - required
  _, err = f.WriteString("    final params = <String, dynamic>{")
  check(err)
  for _, k := range request_keys {
    v := request_map[k]
    is_optional := funk.Reduce(
        v.Comments,
        func (acc bool, c string) bool {
          return acc || strings.HasPrefix(strings.ToLower(strings.Trim(c, " ")), "optional")
        },
        false,
    ).(bool)
    if is_optional == true {
      continue
    }

    _, err = f.WriteString(
      fmt.Sprintf("\n      '%s': %s,", k, makePropName(k)),
    )
    check(err)
  }
  _, err = f.WriteString("\n    };\n")
  check(err)

  // params - optional
  for _, k := range request_keys {
    v := request_map[k]
    is_optional := funk.Reduce(
        v.Comments,
        func (acc bool, c string) bool {
          return acc || strings.HasPrefix(strings.ToLower(strings.Trim(c, " ")), "optional")
        },
        false,
    ).(bool)
    if is_optional == false {
      continue
    }

    _, err = f.WriteString(
      fmt.Sprintf("    if (%[1]s != null)\n      params['%[2]s'] = %[1]s;\n", makePropName(k), k),
    )
    check(err)
  }

  // gen
  param_key := "query_params"
  if ad_info.Method == "post" || ad_info.Method == "put" {
    param_key = "body"
  }
  call_path := strings.Join(
      funk.Map(
        strings.Split(ad_info.Path, "/"),
        func (p string) string {
          if p == "" {
            return ""
          }

          rpp := convertReqParamPair(p)
          if rpp != nil {
            return fmt.Sprintf(
                "$%s",
                rpp[0],
            )
          }

          return p
        },
      ).([]string),
      "/",
  )
  _, err = f.WriteString(
      fmt.Sprintf(
        api_custom_call_api_fmt,
        ad_info.Method,
        call_path,
        param_key,
      ),
  )
  check(err)
  
  _, err = f.WriteString(
      fmt.Sprintf(
        "    final res_json = json.decode(res.body);\n    final result = %s(res_json);\n    await result.init();\n    return result;\n",
        result_class,
      ),
  )
  check(err)


  // tail
  _, err = f.WriteString("  }\n\n")
  check(err)
}

func genCrudApi (
    f *os.File,
    info *table_schema.SchemaInfo,
    crud_type string,
    path string,
    ad_info *ApiDocInfo,
    request_jsonex  *JsonExValue,
    response_jsonex *JsonExValue,
) {
  switch crud_type {
  case "create":
    genCrudApi_create(f, info, path, ad_info, request_jsonex, response_jsonex)
  case "read":
    if strings.Contains(path, "&lt;id&gt;") {
      genCrudApi_getById(f, info, path, ad_info, request_jsonex, response_jsonex)
    }else {
      genCrudApi_get(f, info, path, ad_info, request_jsonex, response_jsonex)
    }
  case "update":
    genCrudApi_update(f, info, path, ad_info, request_jsonex, response_jsonex)
  case "delete":
    genCrudApi_delete(f, info, path, ad_info, request_jsonex, response_jsonex)
  default:
    panic("unknown crud_type " + crud_type)
  }
}

func genCrudApi_delete (
    f *os.File,
    info *table_schema.SchemaInfo,
    path string,
    ad_info *ApiDocInfo,
    request_jsonex  *JsonExValue,
    response_jsonex *JsonExValue,
) {

  // head
  _, err := f.WriteString(
      fmt.Sprintf(
        "  static Future<void> delete_%s (\n    int id,\n    {\n      // null : use WseModel.token\n      // other: use named-token\n      String? token_name\n    }\n  ) async {\n",
        makeFuncNameFromPath(path),
      ),
  )
  check(err)


  // write codes
  _, err = f.WriteString(
      fmt.Sprintf(
        "    await Model.deleteModel(%s.mh, id, user_data: { 'token_name': token_name });\n",
        info.Table_name,
      ),
  )
  check(err)


  // tail
  _, err = f.WriteString("  }\n\n")
  check(err)
}

func genCrudApi_update (
    f *os.File,
    info *table_schema.SchemaInfo,
    path string,
    ad_info *ApiDocInfo,
    request_jsonex  *JsonExValue,
    response_jsonex *JsonExValue,
) {

  // head
  _, err := f.WriteString(
      fmt.Sprintf(
        "  static Future<void> put_%s (\n    int id,\n    {\n      // null : use WseModel.token\n      // other: use named-token\n      String? token_name,\n\n      required dynamic params,\n    }\n  ) async {",
        makeFuncNameFromPath(path),
      ),
  )
  check(err)


  // write codes
  _, err = f.WriteString("    final property_value_map = <Property, dynamic>{};\n")
  check(err)

  for _, sch := range info.Schema {
    if sch.Field == "id" {
      continue
    }
    field := sch.Field
    if field == "#password_hash" {
      field = "password"
    }
    _, err = f.WriteString(
        fmt.Sprintf(
          "    if (params.containsKey('%[2]s'))\n      property_value_map[%[1]s.em.%[3]s] = params['%[2]s'];\n",
          info.Table_name,
          field,
          makePropName(field),
        ),
    )
    check(err)
  }
  _, err = f.WriteString(
      fmt.Sprintf(
        "\n    await %s(id).update(property_value_map, { 'token_name': token_name });\n",
        info.Table_name,
      ),
  )
  check(err)


  // tail
  _, err = f.WriteString("  }\n\n")
  check(err)
}

func genCrudApi_getById (
    f *os.File,
    info *table_schema.SchemaInfo,
    path string,
    ad_info *ApiDocInfo,
    request_jsonex  *JsonExValue,
    response_jsonex *JsonExValue,
) {

  // head
  _, err := f.WriteString(
      fmt.Sprintf(
        "  static Future<%[1]sVM?> get_%[2]s (\n    int id,\n    {\n      // null : use WseModel.token\n      // other: use named-token\n      String? token_name,\n\n      dynamic options,\n      bool?   need_count,\n    }\n  ) async {",
        info.Table_name,
        makeFuncNameFromPath(path),
      ),
  )
  check(err)


  // write codes
  _, err = f.WriteString(
      fmt.Sprintf(api_crud_get_by_id_codes_fmt, info.Table_name),
  )
  check(err)


  // tail
  _, err = f.WriteString("  }\n\n")
  check(err)
}

func genCrudApi_get (
    f *os.File,
    info *table_schema.SchemaInfo,
    path string,
    ad_info *ApiDocInfo,
    request_jsonex  *JsonExValue,
    response_jsonex *JsonExValue,
) {
  // head
  _, err := f.WriteString(
      fmt.Sprintf(
        "  static Future<List<%[1]sVM>> get_%[2]s ({\n      // null : use WseModel.token\n      // other: use named-token\n      String? token_name,\n\n      required dynamic options,\n      dynamic  order_query,\n  }) async {",
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

func genCrudApi_create (
    f *os.File,
    info *table_schema.SchemaInfo,
    path string,
    ad_info *ApiDocInfo,
    request_jsonex  *JsonExValue,
    response_jsonex *JsonExValue,
) {
  var non_model_data_jsonex *JsonExValue
  items := response_jsonex.Value.(map[string]JsonExValue)["items"].Value.([]JsonExValue);
  if len(items) > 0 {
    item := items[0].Value.(map[string]JsonExValue)
    v, has_non_model_data := item["(non_model_data)"]
    if has_non_model_data {
      non_model_data_jsonex = &v
    }
  }

  if non_model_data_jsonex != nil {
    _, err := f.WriteString("  // response has non_model_data\n")
    check(err)
  }

  // head
  _, err := f.WriteString(
      fmt.Sprintf(
        "  static Future<%[1]sVM> post_%[2]s ({\n      // null : use WseModel.token\n      // other: use named-token\n      String? token_name,\n%[3]s\n  }) async {\n",
        info.Table_name,
        makeFuncNameFromPath(path),
        strings.Join(
          funk.Map(
            funk.Filter(
              info.Schema,
              func (sch *table_schema.TableScheme) bool {
                return sch.Field != "id" && sch.Field != "createdAt" && sch.Field != "updatedAt" && sch.Field != "#password_hash";
              },
            ),
            func (sch *table_schema.TableScheme) string {
              field := sch.Field
              if field == "#password_hash" {
                field = "password"
              }
              code := "\n      "
              if sch.Null == false && sch.Default.Valid == false {
                code += "required "
              }
              prop_type := convertTypeFromSql(sch.Type)
              if sch.Null == true {
                code += fmt.Sprintf(
                    "%s? %s",
                    prop_type,
                    makePropName(field),
                )
              }else {
                code += fmt.Sprintf(
                    "%s %s",
                    prop_type,
                    makePropName(field),
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
    func (sch *table_schema.TableScheme) bool {
      return sch.Field != "id" && sch.Field != "createdAt" && sch.Field != "updatedAt" && sch.Null == false
    },
  ).([]*table_schema.TableScheme)
  prop_max_len := funk.Reduce(
    required_schema,
    func (acc int, sch *table_schema.TableScheme) int {
      return funk.MaxInt([]int{acc, len(makePropName(sch.Field))}).(int)
    },
    0,
  ).(int)
  for _, sch := range required_schema {
    field := sch.Field
    if field == "#password_hash" {
      field = "password"
    }
    _, err = f.WriteString(
        fmt.Sprintf(
          "      %[1]s.em.%-[2]*[3]s: %[3]s,\n",
          info.Table_name,
          prop_max_len,
          makePropName(field),
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
  post_on_create_str := ""
  if non_model_data_jsonex != nil {
    _, err = f.WriteString("    var non_model_data = <String, dynamic>{};\n")
    check(err)
    post_on_create_str = "\n          'postOnCreate': (Model m, dynamic res_json) { non_model_data = res_json['(non_model_data)'] as Map<String, dynamic>; },"
  }
  _, err = f.WriteString(
      fmt.Sprintf(
        "    final m = (await Model.createModel(\n        %[1]s.mh,\n        property_value_map,\n        user_data: {\n          'token_name': token_name,%[2]s\n        }\n    )) as %[1]s;\n",
        info.Table_name,
        post_on_create_str,
      ),
  )
  check(err)


  // return view model
  _, err = f.WriteString(
      fmt.Sprintf(
        "    final vm = %sVM({\n",
        info.Table_name,
      ),
  )
  check(err)

  field_max_len := funk.Reduce(
    info.Schema,
    func (acc int, sch *table_schema.TableScheme) int {
      return funk.MaxInt([]int{acc, len(sch.Field)}).(int)
    },
    0,
  ).(int)

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

  if non_model_data_jsonex != nil {
    _, err = f.WriteString(
        fmt.Sprintf(
          "      %-[1]*[2]s: non_model_data,\n",
          field_max_len + 2,
          "'(non_model_data)'",
        ),
    )
    check(err)
  }
  

  // tail
  _, err = f.WriteString("    });\n    await vm.init();\n    return vm;\n  }\n\n")
  check(err)
}

// returns [req_param, uppered_req_param]
func convertReqParamPair (str string) []string {
  subs := re_path_param.FindStringSubmatch(str)
  if len(subs) > 0 {
    req_param := subs[1]
    if req_param == "" {
      req_param = subs[2]
    }
    return []string {
      req_param,
      strings.ToUpper(
        strings.ReplaceAll(strings.ReplaceAll(req_param, "-", ""), "_", ""),
      ),
    }
  }

  return nil
}

func makeModelNameFromPath (path string) string {
  return strings.Join(
      funk.Map(
        strings.Split(path, "/"),
        func (p string) string {
          if p == "" {
            return ""
          }

          rpp := convertReqParamPair(p)
          if rpp != nil {
            return rpp[1]
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
          return r
        },
      ).([]string),
      "_",
  )
}

func makeFuncNameFromPath (path string) string {
  return strings.Join(
      funk.Map(
        strings.Split(path, "/"),
        func (p string) string {
          if p == "" {
            return ""
          }

          rpp := convertReqParamPair(p)
          if rpp != nil {
            return rpp[1]
          }

          r := strings.Join(
            funk.Map(
              strings.Split(p, "-"),
              func (s string) string {
                if len(s) <= 0 {
                  return ""
                }
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

func convertTypeFromDoc (doc_type string) string {
  doc_type = strings.ToUpper(strings.Trim(doc_type, `'"`))
  switch doc_type {
  case "INTEGER", "INT":
    return "int"
  case "DOUBLE":
    return "double"
  case "STRING", "STR", "PASSWORD", "PWD", "TEXT":
    return "String"
  case "BOOLEAN", "BOOL":
    return "bool"
  case "DATETIME", "DATE_TIME", "DATE":
    return "DateTime"
  case "JSON_ARRAY", "ARRAY":
    return "List<dynamic>"
  case "JSON_OBJECT", "JSON_OBJ", "OBJECT":
    return "dynamic"
  default:
    switch {
    case strings.HasPrefix(doc_type, "STRING("):
      return "String"
    case strings.HasPrefix(doc_type, "ENUM"):
      return "String"
    case strings.HasPrefix(doc_type, "FK("):
      return "int"
    }
    panic("unknown type " + doc_type)
  }
}

func getDefaultValueForTypeFromDoc (doc_type string) string {
  switch convertTypeFromDoc(doc_type) {
  case "int":
    return "0"
  case "String":
    return "''"
  case "bool":
    return "false"
  case "DateTime":
    return "DateTime.fromMillisecondsSinceEpoch(0)"
  case "List<dynamic>":
    return "<dynamic>[]"
  case "dynamic":
    return "{}"
  default:
    panic("unknown type for default value - " + doc_type)
  }
}



var re_api_head       = regexp.MustCompile("\n\\#\\# (.*)&nbsp;&nbsp;&nbsp;&nbsp;`(.*)`\n> permission: (.*)\n>.*\n> (.*)\n")
var re_api_request    = regexp.MustCompile(`\n\#\#\# Request` )
var re_api_response   = regexp.MustCompile(`\n\#\#\# Response`)
var re_small_title    = regexp.MustCompile(`\n\#\#\#\# `)
var re_other_statuses = regexp.MustCompile(`\n\#\#\#\# other statuses`)
var re_crud_api       = regexp.MustCompile("CRUD api - `(.*)` of (.*)")
var re_path_param     = regexp.MustCompile("&lt;(.*)&gt;|:(.*)")
var re_md_code        = regexp.MustCompile("```javascript\n((?:.|\\s)*)```\n")


var api_head_str = `
class Api {

`

var api_crud_get_codes_fmt = `
    final res_jsons = await WseModel.find(
        %[1]s.mh,
        options,
        token_name : token_name,
        order_query: order_query,
    );
    final vms = res_jsons.map((e) => %[1]sVM(e)).toList();
    for (final v in vms)
      await v.init();
    return vms;
`

var api_crud_get_by_id_codes_fmt = `
    final res_jsons = await WseModel.findById(
        %[1]s.mh,
        id,
        token_name: token_name,
        options   : options,
        need_count: need_count,
    );
    if (res_jsons.isEmpty)
      return null;

    final vm = %[1]sVM(res_jsons[0]);
    await vm.init();
    return vm;
`

var api_custom_call_api_fmt = `
    var token = WseModel.token;
    if (token_name != null)
      token = WseModel.getNamedToken(token_name);

    final res = await WseApiCall.%s(
      '${WseModel.api_server_address}%s',
      token: token,
      %s: params,
    );
`

type ApiDocInfo struct {
  Path       string
  Method     string
  Permission string
  Comment    string
  Start_idx  int
}
