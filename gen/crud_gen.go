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
        strings.Join(
          funk.Map(schema, func (scheme table_schema.TableScheme) string { return makePropName(scheme.Field) }).([]string),
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

        switch {
        case strings.HasPrefix(scheme.Type, "int("):
          return "int"
        case scheme.Type == "tinyint(1)":
          return "bool"
        case strings.HasPrefix(scheme.Type, "varchar("),
             strings.HasPrefix(scheme.Type, "text"    ):
          return "String"
        case strings.HasPrefix(scheme.Type, "datetime"):
          return "DateTime"
        case strings.HasPrefix(scheme.Type, "enum("):
          return "String"
        default:
          panic("unknown type " + scheme.Type)
        }
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

const model_head_tmp = `
class {{.table_name}} extends WseModel {
  static final _em = {{.table_name}}({{.em_id}});
  static final _handler = {{.table_name}}Modelhandler();

  static {{.table_name}} get em => _em;
  static {{.table_name}}ModelHandler get mh => _handler;


  final {{.id_type}} _id;

`

const model_ctor_fmt = `
  TestModel (this._id) {
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
