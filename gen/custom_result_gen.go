package gen

import (
  "fmt"
  "os"
  "regexp"
  "sort"
  "strings"

  funk "github.com/thoas/go-funk"
  
  table_schema "../table_schema"
)


func GenCustomResult (
    tablename_schemainfo_map map[string]*table_schema.SchemaInfo,
    ad_info *ApiDocInfo,
    response_jsonex *JsonExValue,
) {
  class_name := fmt.Sprintf(
    "%s_%s",
    makeModelNameFromPath(ad_info.Method),
    makeModelNameFromPath(ad_info.Path),
  )

  // create api source code
  f, err := os.Create(
      fmt.Sprintf("./out/custom_results/%s.dart", class_name),
  )
  check(err)
  defer f.Close()


  // import part
  _, err = f.WriteString(
      strings.Join(
        []string{
          "import 'package:wse_mm/wse_model.dart';",
        },
        "\n",
      ),
  )
  check(err)
  _, err = f.WriteString("\n\n")
  check(err)


  // import-model/viewmodel parts
  subresultname_jsonex_map := map[string]JsonExValue{}
  import_model_customresult_list := []string{}
  genImportModelAndCustomResult(&import_model_customresult_list, class_name, tablename_schemainfo_map, "", *response_jsonex, &subresultname_jsonex_map)
  sort.Strings(import_model_customresult_list)
  _, err = f.WriteString(strings.Join(import_model_customresult_list, "\n") + "\n\n")
  check(err)


  // write class
  genCustomResultClass(f, class_name, tablename_schemainfo_map, response_jsonex, true)


  // sub classes
  subresult_names := []string{}
  for sub_name := range subresultname_jsonex_map {
    subresult_names = append(subresult_names, sub_name)
  }
  sort.Strings(subresult_names)
  for _, sub_name := range subresult_names {
    jsonex := subresultname_jsonex_map[sub_name]
    genCustomResultClass(f, sub_name, tablename_schemainfo_map, &jsonex, false)
  }


  // end
  f.Sync()
}

func genCustomResultClass (
    f *os.File,
    class_name string,
    tablename_schemainfo_map map[string]*table_schema.SchemaInfo,
    jsonex *JsonExValue,
    has_message bool,
) {
  if jsonex.Type == "array" {
    arr := jsonex.Value.([]JsonExValue)
    if len(arr) > 0 {
      genCustomResultClass(f, class_name, tablename_schemainfo_map, &arr[0], has_message)
    }
    return
  }

  // write class head
  _, err := f.WriteString(
      fmt.Sprintf(
        "class %s {\n",
        class_name,
      ),
  )
  check(err)


  // properties
  _, err = f.WriteString("  bool _is_inited = false;\n  bool get is_inited => _is_inited;\n\n")
  check(err)
  if has_message {
    _, err = f.WriteString("  String message = '';\n")
    check(err)
  }
  genCustomResultProperties(f, class_name, tablename_schemainfo_map, jsonex)
  _, err = f.WriteString("\n")
  check(err)


  // constructor
  _, err = f.WriteString(
      fmt.Sprintf(
        "  %s (dynamic json) {\n",
        class_name,
      ),
  )
  check(err)
  genSetCustomResultProperties(f, class_name, tablename_schemainfo_map, jsonex)
  _, err = f.WriteString("  }\n\n")
  check(err)


  // init func
  _, err = f.WriteString("  Future<void> init () async {\n")
  check(err)
  genCustomResultInitFunc(f, class_name, tablename_schemainfo_map, jsonex)
  _, err = f.WriteString("  }\n")
  check(err)


  // class tail
  _, err = f.WriteString("}\n\n")
  check(err)
}

func genCustomResultInitFunc (
    f *os.File,
    class_name string,
    tablename_schemainfo_map map[string]*table_schema.SchemaInfo,
    response_jsonex *JsonExValue,
) {
  res_json := response_jsonex.Value.(map[string]JsonExValue)

  res_keys := []string{}
  for k := range res_json {
    res_keys = append(res_keys, k)
  }
  sort.Strings(res_keys)


  // func for check VM class name
  __isVM := func (k string) bool {
    v := res_json[k]

    switch v.Type {
    case "object":
      subs := re_custom_result_model.FindStringSubmatch(strings.Trim(v.Comments[0], " "))
      if len(subs) > 0 {
        model_name := subs[1]
        _, ok := tablename_schemainfo_map[model_name]
        return ok
      }
    case "array":
      elem_type := extractArrayElementTypeFromJsonEx(class_name, tablename_schemainfo_map, k, &v)
      if elem_type[len(elem_type)-2:] != "VM" {
        return false
      }

      _, has_key := tablename_schemainfo_map[elem_type[:len(elem_type)-2]]
      return has_key
    }

    return false
  }


  // gen
  for _, k := range res_keys {
    init_params := ""
    if __isVM(k) == true {
      init_params = "need_fetch: false"
    }

    v := res_json[k]
    switch v.Type {
    case "object":
      _, err := f.WriteString(
          fmt.Sprintf(
            "    await %s!.init(%s);\n",
            makePropName(k),
            init_params,
          ),
      )
      check(err)
    case "array":
      elem_type := extractArrayElementTypeFromJsonEx(class_name, tablename_schemainfo_map, k, &v)
      switch elem_type {
      case "int", "bool", "String", "float", "double":
      default:
        _, err := f.WriteString(
            fmt.Sprintf(
              "    for (final v in %s)\n      await v.init(%s);\n",
              makePropName(k),
              init_params,
            ),
        )
        check(err)
      }
    default:
    }
  }

  _, err := f.WriteString("    _is_inited = true;\n")
  check(err)
}

func genSetCustomResultProperties (
    f *os.File,
    class_name string,
    tablename_schemainfo_map map[string]*table_schema.SchemaInfo,
    response_jsonex *JsonExValue,
) {

  res_json := response_jsonex.Value.(map[string]JsonExValue)

  res_keys := []string{}
  for k := range res_json {
    res_keys = append(res_keys, k)
  }
  sort.Strings(res_keys)

  for _, k := range res_keys {
    v := res_json[k]
    _, err := f.WriteString(
        fmt.Sprintf(
          "    if (json.containsKey('%s') && json['%s'] != null) {\n",
          k,
          k,
        ),
    )
    check(err)

    switch v.Type {
    case "object":
      obj_name := fmt.Sprintf("%s__%s", class_name, k)
      if len(v.Comments) > 0 {
        subs := re_custom_result_model.FindStringSubmatch(strings.Trim(v.Comments[0], " "))
        if len(subs) > 0 {
          model_name := subs[1]
          if _, ok := tablename_schemainfo_map[model_name]; ok {
            // model
            obj_name = model_name + "VM"
            _, err := f.WriteString(
                fmt.Sprintf(
                  "      WseModel.registerByJson(%s.mh, json['%s']);\n",
                  model_name,
                  k,
                ),
            )
            check(err)
          }
        }
      }
      
      _, err := f.WriteString(
          fmt.Sprintf(
            "      %s = %s(json['%s']);\n",
            makePropName(k),
            obj_name,
            k,
          ),
      )
      check(err)
    case "array":
      _, err := f.WriteString(
          fmt.Sprintf(
            "      for (final v in json['%s'] as List<dynamic>) {\n",
            k,
          ),
      )
      check(err)
      elem_type := extractArrayElementTypeFromJsonEx(class_name, tablename_schemainfo_map, k, &v)
      switch elem_type {
      case "dynamic":
        _, err = f.WriteString(
            fmt.Sprintf(
              "        %s.add(v);\n",
              makePropName(k),
            ),
        )
        check(err)
      case "int", "bool", "String", "float", "double":
        _, err = f.WriteString(
            fmt.Sprintf(
              "        %s.add(v as %s);\n",
              makePropName(k),
              elem_type,
            ),
        )
        check(err)
      default:
        if len(elem_type) > 2 && elem_type[len(elem_type)-2:] == "VM" {
          model_name := elem_type[:len(elem_type)-2]
          if _, ok := tablename_schemainfo_map[model_name]; ok {
            // model
            _, err = f.WriteString(
                fmt.Sprintf(
                  "        WseModel.registerByJson(%s.mh, v);\n",
                  model_name,
                ),
            )
            check(err)
          }
        }
        _, err = f.WriteString(
            fmt.Sprintf(
              "        %s.add(%s(v));\n",
              makePropName(k),
              elem_type,
            ),
        )
        check(err)
      }
      _, err = f.WriteString("      }\n")
      check(err)
    default:
      type_str := convertTypeFromDoc(v.Value.(string))
      switch type_str {
      case "DateTime":
        _, err := f.WriteString(
            fmt.Sprintf(
              "      %s = DateTime.parse(json['%s'] as String);\n",
              makePropName(k),
              k,
            ),
        )
        check(err)
      default:
        _, err := f.WriteString(
            fmt.Sprintf(
              "      %s = json['%s'] as %s;\n",
              makePropName(k),
              k,
              convertTypeFromDoc(v.Value.(string)),
            ),
        )
        check(err)
      }
    }
    _, err = f.WriteString("    }\n")
    check(err)
  }
}

func genCustomResultProperties (
    f *os.File,
    class_name string,
    tablename_schemainfo_map map[string]*table_schema.SchemaInfo,
    response_jsonex *JsonExValue,
) {
  res_json := response_jsonex.Value.(map[string]JsonExValue)

  res_keys := []string{}
  for k := range res_json {
    res_keys = append(res_keys, k)
  }
  sort.Strings(res_keys)

  for _, k := range res_keys {
    v := res_json[k]
    if k == "message" {
      continue
    }
    genCustomResultSingleProperty(f, class_name, tablename_schemainfo_map, k, &v)
  }
}

func genCustomResultSingleProperty (
    f *os.File,
    class_name string,
    tablename_schemainfo_map map[string]*table_schema.SchemaInfo,
    k string,
    v *JsonExValue,
) {
  // prepare for optional
  is_optional := funk.Reduce(
      v.Comments,
      func (acc bool, c string) bool {
        return acc || strings.HasPrefix(strings.ToLower(strings.Trim(c, " ")), "optional")
      },
      false,
  ).(bool)
  optional_chr := ""
  default_value := ""
  if is_optional {
    optional_chr = "?"
  }


  // parse
  switch v.Type {
  case "object":
    obj_name := fmt.Sprintf("%s__%s", class_name, k)
    if len(v.Comments) > 0 {
      subs := re_custom_result_model.FindStringSubmatch(strings.Trim(v.Comments[0], " "))
      if len(subs) > 0 {
        model_name := subs[1]
        if _, ok := tablename_schemainfo_map[model_name]; ok {
          // model
          obj_name = model_name + "VM"
        }
      }
    }
    optional_chr = "?"
    _, err := f.WriteString(
        fmt.Sprintf(
          "  %s%s %s%s;\n",
          obj_name,
          optional_chr,
          makePropName(k),
          default_value,
        ),
    )
    check(err)
  case "array":
    elem_type := extractArrayElementTypeFromJsonEx(class_name, tablename_schemainfo_map, k, v)
    _, err := f.WriteString(
        fmt.Sprintf(
          "  final %s = <%s>[];\n",
          makePropName(k),
          elem_type,
        ),
    )
    check(err)
  default:
    if is_optional == false {
      default_value = fmt.Sprintf(" = %s", getDefaultValueForTypeFromDoc(v.Value.(string)))
    }
    _, err := f.WriteString(
        fmt.Sprintf(
          "  %s%s %s%s;\n",
          convertTypeFromDoc(v.Value.(string)),
          optional_chr,
          makePropName(k),
          default_value,
        ),
    )
    check(err)
  }
}

func extractArrayElementTypeFromJsonEx (
    class_name string,
    tablename_schemainfo_map map[string]*table_schema.SchemaInfo,
    key string,
    jsonex_arr *JsonExValue,
) string {
  arr := jsonex_arr.Value.([]JsonExValue)
  if len(arr) <= 0 {
    return "dynamic"
  }

  switch arr[0].Type {
  case "":
    return "dynamic"
  case "object":
    obj_name := fmt.Sprintf("%s__%s", class_name, key)
    if len(arr[0].Comments) > 0 {
      subs := re_custom_result_model.FindStringSubmatch(strings.Trim(arr[0].Comments[0], " "))
      if len(subs) > 0 {
        model_name := subs[1]
        if _, ok := tablename_schemainfo_map[model_name]; ok {
          // model
          obj_name = model_name + "VM"
        }
      }
    }
    return obj_name
  case "array":
    return fmt.Sprintf(
        "List<%s>",
        extractArrayElementTypeFromJsonEx(class_name, tablename_schemainfo_map, key, &arr[0]),
    )
  default:
    return convertTypeFromDoc(arr[0].Value.(string))
  }
}

func genImportModelAndCustomResult(
    result *[]string,
    class_name string,
    tablename_schemainfo_map map[string]*table_schema.SchemaInfo,
    key string,
    jsonex JsonExValue,
    subresultname_jsonex_map *map[string]JsonExValue,
) {
  switch jsonex.Type {
  case "object":
    if len(jsonex.Comments) > 0 {
      subs := re_custom_result_model.FindStringSubmatch(strings.Trim(jsonex.Comments[0], " "))
      if len(subs) > 0 {
        obj_name := subs[1]
        if _, ok := tablename_schemainfo_map[obj_name]; ok {
          // model
          *result = append(
              *result,
              fmt.Sprintf(
                "import '../models/%[1]s.dart';\nimport '../view_models/%[1]sVM.dart';",
                obj_name,
              ),
          )
          break
        }
      }
    }

    // non-model
    if key != "" {
      obj_name := fmt.Sprintf("%s__%s", class_name, key)
      (*subresultname_jsonex_map)[obj_name] = jsonex
    }
    for k, v := range jsonex.Value.(map[string]JsonExValue) {
      genImportModelAndCustomResult(result, class_name, tablename_schemainfo_map, k, v, subresultname_jsonex_map)
    }
  case "array":
    arr := jsonex.Value.([]JsonExValue)
    if len(arr) > 0 {
      genImportModelAndCustomResult(result, class_name, tablename_schemainfo_map, key, arr[0], subresultname_jsonex_map)
    }
  }
}



var re_custom_result_model = regexp.MustCompile("Model:(.*)")
