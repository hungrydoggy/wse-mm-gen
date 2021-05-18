package gen

import (
  "bytes"
  "fmt"
  "os"
  "regexp"
  "strings"
	"text/template"

  funk "github.com/thoas/go-funk"

  table_schema "../table_schema"
)

func GenCodeForCrud (
    table_name string,
    schema []table_schema.TableScheme,
    manyname_modelname_map map[string]string,
) {
  genModel(table_name, schema, manyname_modelname_map, "id", "int", "0")
  genViewModel(table_name, schema, manyname_modelname_map, "id", "int", "0")
}

func genViewModel (
    table_name string,
    schema []table_schema.TableScheme,
    manyname_modelname_map map[string]string,
    id_key string,
    id_type string,
    em_id string,
) {
  vm_name := table_name + "VM"

  f, err := os.Create(fmt.Sprintf("./out/view_models/%s.dart", vm_name))
  check(err)
  defer f.Close()

  // import default
  _, err = f.WriteString(vm_import_str)
  check(err)


  // import model
  _, err = f.WriteString(
      fmt.Sprintf("import '../models/%s.dart';\n\n", table_name),
  )
  check(err)


  // import view models
  viewmodel_check_map := map[string]bool{}
  for _, sch := range schema {
    if sch.FieldType != table_schema.ASSOCIATION {
      continue
    }
    viewmodel_check_map[sch.Association_info.Model_name + "VM"] = true
  }
  for _, model_name := range manyname_modelname_map {
    viewmodel_check_map[model_name + "VM"] = true
  }
  for view_model := range viewmodel_check_map {
    if view_model == table_name + "VM" {
      continue
    }
    _, err = f.WriteString(
        fmt.Sprintf("import './%s.dart';\n", view_model),
    )
    check(err)
  }
  _, err = f.WriteString("\n\n")
  check(err)


  // view model head
  _, err = f.WriteString(
      fmt.Sprintf("class %s extends ViewModel {\n  %s _id = %s;\n  int get id => _id;\n\n", vm_name, id_type, em_id),
  )
  check(err)


  // normal properties
  prop_type_max_len := int(
      funk.Reduce(
        schema,
        func (acc int, sch table_schema.TableScheme) int {
          return funk.MaxInt([]int{acc, len(convertTypeFromSql(sch.Type))}).(int)
        },
        0,
      ),
  )
  for _, sch := range schema {
    if sch.Field == id_key {
      continue;
    }
    _, err = f.WriteString(
        fmt.Sprintf(
          "  VMProperty<%-[1]*[2]s>? _%[3]s;\n",
          prop_type_max_len,
          convertTypeFromSql(sch.Type),
          makePropName(sch.Field),
        ),
    )
    check(err)
  }
  _, err = f.WriteString("\n")
  check(err)


  // normal properties getter
  prop_max_len := int(
      funk.Reduce(
        schema,
        func (acc int, sch table_schema.TableScheme) int {
          return funk.MaxInt([]int{acc, len(makePropName(sch.Field))}).(int)
        },
        0,
      ),
  )
  for _, sch := range schema {
    if sch.Field == id_key {
      continue;
    }
    _, err = f.WriteString(
        fmt.Sprintf(
          "  VMProperty<%-[1]*[2]s>? get %-[3]*[4]s => _%[4]s;\n",
          prop_type_max_len,
          convertTypeFromSql(sch.Type),
          prop_max_len,
          makePropName(sch.Field),
        ),
    )
    check(err)
  }
  _, err = f.WriteString("\n")
  check(err)


  // fk properties
  fk_model_max_len := int(
      funk.Reduce(
        schema,
        func (acc int, sch table_schema.TableScheme) int {
          str_len := 0
          if sch.FieldType == table_schema.ASSOCIATION {
            str_len = len(sch.Association_info.Model_name)
          }
          return funk.MaxInt([]int{acc, str_len}).(int)
        },
        0,
      ),
  )
  for _, sch := range schema {
    if sch.FieldType != table_schema.ASSOCIATION {
      continue
    }

    info := sch.Association_info
    _, err = f.WriteString(
        fmt.Sprintf(
          "  %-[1]*[2]s _%[3]s;\n",
          fk_model_max_len + 3,
          info.Model_name + "VM?",
          info.As_name,
        ),
    )
    check(err)
  }
  _, err = f.WriteString("\n")
  check(err)


  // fk properties getter
  as_name_max_len := int(
      funk.Reduce(
        schema,
        func (acc int, sch table_schema.TableScheme) int {
          str_len := 0
          if sch.FieldType == table_schema.ASSOCIATION {
            str_len = len(sch.Association_info.As_name)
          }
          return funk.MaxInt([]int{acc, str_len}).(int)
        },
        0,
      ),
  )
  for _, sch := range schema {
    if sch.FieldType != table_schema.ASSOCIATION {
      continue
    }

    info := sch.Association_info
    _, err = f.WriteString(
        fmt.Sprintf(
          "  %-[1]*[2]s get %-[3]*[4]s => _%[4]s;\n",
          fk_model_max_len + 3,
          info.Model_name + "VM?",
          as_name_max_len,
          info.As_name,
        ),
    )
    check(err)
  }
  _, err = f.WriteString("\n")
  check(err)


  // has-many properties
  model_name_max_len := 0
  for _, model_name := range manyname_modelname_map {
    model_name_max_len = funk.MaxInt([]int{model_name_max_len, len(model_name)}).(int)
  }
  for many_name, model_name := range manyname_modelname_map {
    _, err = f.WriteString(
        fmt.Sprintf(
          "  List<%[1]*[2]s>? _%[3]s;\n",
          model_name_max_len + 2,
          model_name + "VM",
          makePropName(many_name),
        ),
    )
    check(err)
  }
  _, err = f.WriteString("\n")
  check(err)


  // has-many properties getter
  many_name_max_len := 0
  for many_name := range manyname_modelname_map {
    many_name_max_len = funk.MaxInt([]int{many_name_max_len, len(many_name)}).(int)
  }
  for many_name, model_name := range manyname_modelname_map {
    _, err = f.WriteString(
        fmt.Sprintf(
          "  List<%[1]*[2]s>? get %[3]*[4]s => _%[4]s;\n",
          model_name_max_len + 2,
          model_name + "VM",
          many_name_max_len,
          makePropName(many_name),
        ),
    )
    check(err)
  }
  if len(manyname_modelname_map) > 0 {
    _, err = f.WriteString("\n\n")
    check(err)
  }


  // constructor
  genVMConstructor(f, table_name, vm_name, schema, manyname_modelname_map, id_key, id_type)

  // view model tail
  _, err = f.WriteString("}")
  check(err)


  // end
  f.Sync()
}

func genVMConstructor(
    f *os.File,
    table_name string,
    vm_name string,
    schema []table_schema.TableScheme,
    manyname_modelname_map map[string]string,
    id_key string,
    id_type string,
) {
  // head
  _, err := f.WriteString(
      fmt.Sprintf(
        "  %s (dynamic json, {String? vm_name}): super(vm_name: vm_name) {\n    if (json.containsKey('id') == false)\n      throw 'no id';\n\n    _id = json['id'];\n\n\n",
        vm_name,
      ),
  )
  check(err)


  // properties
  _, err = f.WriteString("    // set properties\n    final properties = <VMProperty>[];\n")
  check(err)

  for _, sch := range schema {
    if sch.Field == id_key {
      continue
    }
    _, err = f.WriteString(
        fmt.Sprintf(
          "    if (json.containsKey('%[1]s')) {\n      _%[2]s = VMProperty<%[3]s>(%[4]s.mh, _id, '%[1]s');\n      properties.add(%[2]s!);\n    }\n\n",
          sch.Field,
          makePropName(sch.Field),
          convertTypeFromSql(sch.Type),
          table_name,
        ),
    )
    check(err)
  }

  _, err = f.WriteString("    setProperties(properties);\n\n\n")
  check(err)


  // associations
  _, err = f.WriteString("    // set nested vms\n    final nested_vms = <ViewModel>[];\n")
  check(err)

  for _, sch := range schema {
    if sch.FieldType != table_schema.ASSOCIATION {
      continue
    }
    _, err = f.WriteString(
        fmt.Sprintf(
          "    if (json.containsKey('*%[1]s')) {\n      _%[1]s = %[2]s(json['*%[1]s'], vm_name: '*%[1]s');\n      nested_vms.add(_%[1]s!);\n    }\n\n",
          sch.Association_info.As_name,
          sch.Association_info.Model_name + "VM",
        ),
    )
    check(err)
  }


  // has-many associations
  for many_name, model_name := range manyname_modelname_map {
    _, err = f.WriteString(
        fmt.Sprintf(
          "    if (json.containsKey('*%s')) {\n",
          many_name,
        ),
    )
    check(err)

    _, err = f.WriteString(
        fmt.Sprintf(
          "      var ni = 0;\n      for (final nested_json in json['*%[1]s']) {\n        final vm = %[2]s(nested_json, vm_name: '*%[1]s.' + ni.toString());\n        _%[1]s!.add(vm);\n        nested_vms.add(vm);\n        ni += 1;\n      }\n\n",
          many_name,
          model_name + "VM",
        ),
    )
    check(err)

    _, err = f.WriteString("    }\n\n")
    check(err)
  }

  _, err = f.WriteString("    setNestedVMs(nested_vms);\n")
  check(err)


  // end of constructor
  _, err = f.WriteString("  }\n")
  check(err)
}

func genModel (
    table_name string,
    schema []table_schema.TableScheme,
    manyname_modelname_map map[string]string,
    id_key string,
    id_type string,
    em_id string,
) {
  f, err := os.Create(fmt.Sprintf("./out/models/%s.dart", table_name))
  check(err)
  defer f.Close()

  // import default
  _, err = f.WriteString(import_str)
  check(err)


  /// import other models
  othermodelname_check_map := map[string]bool{}

  // from association
  for _, sch := range schema {
    if sch.FieldType == table_schema.ASSOCIATION {
      othermodelname_check_map[sch.Association_info.Model_name] = true
    }
  }

  // from many-name
  for _, model_name := range manyname_modelname_map {
    othermodelname_check_map[model_name] = true
  }

  // generate
  for model_name := range othermodelname_check_map {
    if model_name == table_name {
      continue
    }

    _, err = f.WriteString(
        fmt.Sprintf("import './%s.dart';\n", model_name),
    )
    check(err)
  }
  _, err = f.WriteString("\n\n")
  check(err)


  /// model
  // model head
  _, err = f.WriteString(
      Tprintf(
        "model_head",
        model_head_tmp,
        map[string]interface{}{
          "table_name": table_name,
          "id_type"   : id_type,
          "em_id"     : em_id,
        },
      ),
  )
  check(err)

  // properties
  genProperties(f, table_name, schema, id_key)

  // constructor
  _, err = f.WriteString(
      fmt.Sprintf(
        model_ctor_fmt,
        table_name,
        strings.Join(
          funk.Filter(
            funk.Map(
              schema,
              func (scheme table_schema.TableScheme) string { return makePropName(scheme.Field) },
            ),
            func (e string) bool {
              return e != "id"
            },
          ).([]string),
          ", \n      ",
        ) + ",",
      ),
  )
  check(err)

  // model tail
  _, err = f.WriteString(model_tail_str)
  check(err)



  /// model handler
  // make key_nestedhandler_str
  key_nestedhandler_str := "{"
  associations := funk.Filter(
      schema,
      func (sch table_schema.TableScheme) bool { return sch.FieldType == table_schema.ASSOCIATION; },
  ).([]table_schema.TableScheme)
  for _, ass := range associations {
    info := ass.Association_info
    key_nestedhandler_str += fmt.Sprintf("\n    '*%s': %s.mh,", info.As_name, info.Model_name)
  }
  for many_name, model_name := range manyname_modelname_map {
    key_nestedhandler_str += fmt.Sprintf("\n    '*%s': %s.mh,", many_name, model_name)
  }
  if key_nestedhandler_str != "{" {
    key_nestedhandler_str += "\n  "
  }
  key_nestedhandler_str += "}"

  // write code
  _, err = f.WriteString(
      Tprintf(
        "model_handler",
        model_handler_tmp,
        map[string]interface{}{
          "table_name"       : table_name,
          "path"             : makePathName(table_name),
          "id_key"           : id_key,
          "key_nestedhandler": key_nestedhandler_str,
        },
      ),
  )
  check(err)


  // end
  f.Sync()
}


var re_first_cap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var re_all_cap   = regexp.MustCompile("([a-z0-9])([A-Z])")
func makePathName (table_name string) string {
	path := re_first_cap.ReplaceAllString(table_name, "${1}_${2}")
	path  = re_all_cap  .ReplaceAllString(path      , "${1}_${2}")
	path = strings.ToLower(path)

  switch {
  case path[len(path)-1] == 's',
       path[len(path)-1] == 'x',
       path[len(path)-1] == 'o',
       path[len(path)-2:] == "ch",
       path[len(path)-2:] == "sh":
    path += "es"
  case path[len(path)-1] == 'y':
    path = path[:len(path)-1] + "ies"
  case path[len(path)-1] == 'f':
    path = path[:len(path)-1] + "ves"
  case path[len(path)-2:] == "fe":
    path = path[:len(path)-2] + "ves"
  default:
    path += "s"
  }

  return path
}

func convertTypeFromSql (sql_type string) string {
  switch {
  case strings.HasPrefix(sql_type, "int("):
    return "int"
  case sql_type == "tinyint(1)":
    return "bool"
  case strings.HasPrefix(sql_type, "varchar("),
       strings.HasPrefix(sql_type, "text"    ):
    return "String"
  case strings.HasPrefix(sql_type, "datetime"):
    return "DateTime"
  case strings.HasPrefix(sql_type, "enum("):
    return "String"
  default:
    panic("unknown type " + sql_type)
  }
}

func genProperties (f *os.File, table_name string, schema []table_schema.TableScheme, id_key string) {
  if len(schema) <= 0 {
    return
  }

  // compute field_max_len
  field_max_len := funk.Reduce(
      schema,
      func (acc int, scheme table_schema.TableScheme) int {
        return funk.MaxInt([]int{acc, len(makePropName(scheme.Field))}).(int)
      },
      0,
  )

  // convert types
  converted_types := funk.Map(
      schema,
      func (scheme table_schema.TableScheme) string {
        if scheme.Field == id_key {
          return ""
        }
        return convertTypeFromSql(scheme.Type)
      },
  ).([]string)
  converted_type_max_len := funk.Reduce(
      converted_types,
      func (acc int, ct string) int {
        return funk.MaxInt([]int{acc, len(ct)}).(int)
      },
      0,
  )

  // write code
  for si, scheme := range schema {
    if scheme.Field == id_key {
      continue
    }

    _, err := f.WriteString(
        fmt.Sprintf(
          "  final %-[1]*[2]s = Property<%-[3]*[4]s>(name: %-[5]*[6]s);\n",
          int(field_max_len),
          makePropName(scheme.Field),
          int(converted_type_max_len),
          converted_types[si],
          int(field_max_len),
          "'"+scheme.Field+"'",
        ),
    )
    check(err)
  }
}

func makePropName (field_name string) string {
  switch field_name[0] {
  case '@':
    return "fk_" + field_name[1:]
  case '#':
    return "hd_" + field_name[1:]
  default:
    return field_name
  }
}

func check(e error) {
  if e != nil {
    panic(e)
  }
}

func Tprintf (template_name string, template_str string, data map[string]interface{}) string {
  t := template.Must(template.New(template_name).Parse(template_str))
  buf := &bytes.Buffer{}
  if err := t.Execute(buf, data); err != nil {
    check(err)
  }
  return buf.String()
}

const import_str = `import 'package:mm/model.dart';
import 'package:mm/property.dart';
import 'package:wse_mm/wse_model.dart';

`

const vm_import_str = `import 'package:mm/view_model.dart';
import 'package:mm/vm_property.dart';

`

const model_head_tmp = `
class {{.table_name}} extends WseModel {
  static final _em = {{.table_name}}({{.em_id}});
  static final _handler = {{.table_name}}ModelHandler();

  static {{.table_name}} get em => _em;
  static {{.table_name}}ModelHandler get mh => _handler;


  final {{.id_type}} _id;

`

const model_ctor_fmt = `
  %s (this._id) {
    setProperties([
      %s
    ]);
  }
`

const model_tail_str = `
  @override
  int get id => _id;

  @override
  String get model_name => _handler.model_name;

  @override
  ModelHandler get handler => _handler;

}

`

const model_handler_tmp = `
class {{.table_name}}ModelHandler extends WseModelHandler {
  @override
  String get path => '{{.path}}';

  @override
  String get id_key => '{{.id_key}}';

  @override
  String get model_name => '{{.table_name}}';

  @override
  Model newInstance (id) => {{.table_name}}(id);

  @override
  Map<String, WseModelHandler> get key_nestedhandler => {{.key_nestedhandler}};
}`
