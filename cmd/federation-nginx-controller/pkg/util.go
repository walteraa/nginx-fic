package controller

import(
  "os"
  "io/ioutil"
  "text/template"
)

const (
  nginxCfg = "nginx.conf"
)

var(
  nginxTmpl = "nginx.tmpl"
)

func WriteCfg(ingressName string, paths []IngressPath) ( string, error) {
  file,err := os.Create(ingressName + "_" + nginxCfg)
  
  if err != nil{
    return "",err
  }

  defer file.Close()

  conf := make(map[string] interface{})

  conf["paths"] = paths

  tmpl,err := template.ParseFiles(nginxTmpl)
  
  if err != nil {
    return "",err
  }

  err = tmpl.Execute(file, conf)
 
  if err != nil{
    return "",err
  }
  buffer,err := ioutil.ReadFile(ingressName + "_" + nginxCfg)

  if err != nil{
    return "",err
  }

  data := string(buffer)


  return data,nil
}
func DeleteCfg(ingressName string) error{
  err := os.Remove(ingressName + "_"+nginxCfg)
  return err
}
