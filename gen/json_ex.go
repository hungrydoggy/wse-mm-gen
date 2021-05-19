package gen

import (
  //"fmt"
  "regexp"
  "strings"
)


func ParseJsonEx (str string) JsonExValue {
  str = strings.Trim(str, blank_cut_set)
  
  for idx:=0; true; idx+=1 {
    switch str[idx] {
    case '{':
      res, idx := parseJsonEx_obj(str, idx)
      left := strings.Trim(str[idx+1:], blank_cut_set)
      if len(left) > 0 {
        ei := 30
        if ei > len(left) {
          ei = len(left)
        }
        panic("parse json-ex err: left error - " + left[:ei])
      }
      return res
    case '[':
      res, idx := parseJsonEx_arr(str, idx)
      left := strings.Trim(str[idx+1:], blank_cut_set)
      if len(left) > 0 {
        ei := 30
        if ei > len(left) {
          ei = len(left)
        }
        panic("parse json-ex err: left error - " + left[:ei])
      }
      return res
    case '/':
      idxes := re_jsonex_comment.FindStringSubmatchIndex(str[idx:])
      idx += idxes[1]
    default:
      panic("json-ex must start with [ or {")
    }
  }

  return JsonExValue{}
}

func parseJsonEx_arr (str string, idx int) (JsonExValue, int) {
  result := []JsonExValue{}


  // parse arr_comments
  arr_comments := []string{}
  for idx+=1; idx<len(str); idx+=1 {
    if strings.Contains(blank_cut_set, str[idx:idx+1]) {
      continue
    }
    if str[idx] != '/' {
      idx -= 1
      break
    }

    sub_idxes := re_jsonex_comment.FindStringSubmatchIndex(str[idx:])
    if len(sub_idxes) <= 0 {
      panicJsonEx("syntax err", str, idx)
    }
    arr_comments = append(arr_comments, str[idx+sub_idxes[2]: idx+sub_idxes[3]])
    idx += sub_idxes[1]
  }


  // parse
  comments := []string{}
  is_element_set := false
  element_str := ""
  for idx+=1; idx<len(str); idx+=1 {
    if strings.Contains(blank_cut_set, str[idx:idx+1]) {
      continue
    }

    switch str[idx] {
    case '/':
      sub_idxes := re_jsonex_comment.FindStringSubmatchIndex(str[idx:])
      if len(sub_idxes) <= 0 {
        panicJsonEx("syntax err", str, idx)
      }
      comments = append(comments, str[idx+sub_idxes[2]: idx+sub_idxes[3]])
      idx += sub_idxes[1]
    case ']':
      return JsonExValue{"array", result, arr_comments}, idx
    case '[':
      if is_element_set == true {
        panicJsonEx("element is already set", str, idx)
      }
      arr, ridx := parseJsonEx_arr(str, idx)
      result = append(result, arr)
      is_element_set = true
      idx = ridx
    case '{':
      if is_element_set == true {
        panicJsonEx("element is already set", str, idx)
      }
      obj, ridx := parseJsonEx_obj(str, idx)
      result = append(result, obj)
      is_element_set = true
      idx = ridx
    case '"':
      if is_element_set == true {
        panicJsonEx("syntax error", str, idx)
      }
      sub_idxes := re_string.FindStringSubmatchIndex(str[idx:])
      element := str[idx+sub_idxes[2]:idx+sub_idxes[3]]
      if strings.Contains(element, "\n") {
        panicJsonEx("invalid string", str, idx)
      }
      result = append(result, JsonExValue{"string", `"` + element + `"`, comments})
      is_element_set = true
    case ',':
      if is_element_set == false {
        result = append(result, JsonExValue{"string", element_str, comments})
      }
      element_str = ""
      is_element_set = false
      comments = []string{}
    default:
      if is_element_set == true {
        panicJsonEx("element is already set", str, idx)
      }
      element_str += str[idx:idx+1]
    }
  }

  return JsonExValue{"array", result, arr_comments}, idx
}

func parseJsonEx_obj (str string, idx int) (JsonExValue, int) {
  result := map[string]JsonExValue{}


  // parse obj_comments
  obj_comments := []string{}
  for idx+=1; idx<len(str); idx+=1 {
    if strings.Contains(blank_cut_set, str[idx:idx+1]) {
      continue
    }
    if str[idx] != '/' {
      idx -= 1
      break
    }

    sub_idxes := re_jsonex_comment.FindStringSubmatchIndex(str[idx:])
    if len(sub_idxes) <= 0 {
      panicJsonEx("syntax err", str, idx)
    }
    obj_comments = append(obj_comments, str[idx+sub_idxes[2]: idx+sub_idxes[3]])
    idx += sub_idxes[1]
  }
  

  // parse
  comments := []string{}
  is_key_set   := false
  is_value_set := false
  key := ""
  value_str := ""
  for idx+=1; idx<len(str); idx+=1 {
    if strings.Contains(blank_cut_set, str[idx:idx+1]) {
      continue
    }

    switch str[idx] {
    case '/':
      sub_idxes := re_jsonex_comment.FindStringSubmatchIndex(str[idx:])
      if len(sub_idxes) <= 0 {
        panicJsonEx("syntax err", str, idx)
      }
      comments = append(comments, str[idx+sub_idxes[2]: idx+sub_idxes[3]])
      idx += sub_idxes[1]
    case '}':
      if is_key_set == true && is_value_set == false {
        panicJsonEx("value must be set", str, idx)
      }
      return JsonExValue{"object", result, obj_comments}, idx
    case '{':
      if is_key_set == false {
        panicJsonEx("key must be set first before", str, idx)
      }else if is_value_set == true {
        panicJsonEx("value is already set", str, idx)
      }
      obj, ridx := parseJsonEx_obj(str, idx)
      result[key] = obj
      is_value_set = true
      idx = ridx
    case '[':
      if is_key_set == false {
        panicJsonEx("key must be set first before", str, idx)
      }else if is_value_set == true {
        panicJsonEx("value is already set", str, idx)
      }
      arr, ridx := parseJsonEx_arr(str, idx)
      result[key] = arr
      is_value_set = true
      idx = ridx
    case '"':
      if is_key_set == false {
        sub_idxes := re_obj_key.FindStringSubmatchIndex(str[idx:])
        key = str[idx+sub_idxes[2]:idx+sub_idxes[3]]
        if strings.Contains(key, "\n") {
          panicJsonEx("invalid string", str, idx)
        }
        idx += sub_idxes[4] - 1
        is_key_set = true
      }else if is_value_set == false {
        sub_idxes := re_string.FindStringSubmatchIndex(str[idx:])
        value := str[idx+sub_idxes[2]:idx+sub_idxes[3]]
        if strings.Contains(value, "\n") {
          panicJsonEx("invalid string", str, idx)
        }
        result[key] = JsonExValue{"string", `"` + value + `"`, comments}
        is_value_set = true
      }else {
        panicJsonEx("syntax error", str, idx)
      }
    case ',':
      if is_key_set == false {
        panicJsonEx("key must be set first before", str, idx)
      }
      if is_value_set == false {
        result[key] = JsonExValue{"string", value_str, comments}
      }
      key       = ""
      value_str = ""
      is_key_set   = false
      is_value_set = false
      comments = []string{}
    case ':':
      if is_key_set == true {
        panicJsonEx("key is already set", str, idx)
      }
      is_key_set = true
    default:
      if is_key_set == false {
        key += str[idx:idx+1]
        continue
      }else if is_value_set == true {
        panicJsonEx("value is already set", str, idx)
      }
      value_str += str[idx:idx+1]
    }
  }

  return JsonExValue{"object", result, obj_comments}, idx
}

func panicJsonEx (panic_str string, str string, idx int) {
  si := idx
  ei := idx + 30
  if ei > len(str) {
    ei = len(str)
  }
  panic("parse json-ex err: " + panic_str + " - " + str[si:ei] + "...")
}

var blank_cut_set = " \t\r\n"

var re_string         = regexp.MustCompile(`[^\\]?"((?:\\"|[^"])*?)"`)
var re_obj_key        = regexp.MustCompile(`[^\\]?"((?:\\"|[^"])*?)"\s*:\s*(.)`)
var re_jsonex_comment = regexp.MustCompile(`//(.*)\n`)


type JsonExValue struct {
  Type     string
  Value    interface{}
  Comments []string
}
