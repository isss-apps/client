package main

import (
  "os"
  "fmt"
  "errors"
  "bytes"
  "net/http"
  "encoding/json"
  "io/ioutil"
)

type item struct {
  CatalogId int  `json:"catalogId"`
  Amount int     `json:"amount"`
}

type order struct {
  Name string     `json:"name"`
  Address string  `json:"address"`
  Items []item    `json:"items"`
}

type action struct {
  action int
  badRequest bool
}

type connection struct {
  Url string `json:"url"`
  Port int   `json:"port"`
}

const (
  QUERY_ITEMS = iota
  QUERY_ITEM_AVAILABILITY
  ORDER_ITEM
)

func parseOperation(op string) int {
  var opResult int
  switch op {
  case "items":
    opResult = QUERY_ITEMS
    break
  case "availability":
    opResult = QUERY_ITEM_AVAILABILITY
    break
  case "order":
    opResult = ORDER_ITEM
    break
  default:
    opResult = -1
  }
  return opResult
}

func checkArgs() (*action, error) {
  switch len(os.Args) {
  case 2:
    return &action{parseOperation(os.Args[1]), false}, nil
  case 3:
    if os.Args[2] == "--error" {
      return &action{parseOperation(os.Args[1]), true}, nil
    }
  }
  return nil, errors.New(getHelp())
}

func queryItems(config *connection) error {
  client := &http.Client{}
  for categoty, _ := range []int{0,1,2} {
    url := fmt.Sprintf("http://%s:%d/catalog/list/%d", config.Url, config.Port, categoty)
    request, err := http.NewRequest("GET", url, nil)
    if err != nil {
      return err
    }
    request.Header.Set("Accept", "application/json")
    response, err := client.Do(request)
    if err != nil {
      return err
    }
    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
      return err
    }
    prettyPrintJsonBody(body)
    response.Body.Close()
  }
  return nil
}

func queryAvailability(config *connection) error {
  url := fmt.Sprintf("http://%s:%d/availability/5", config.Url, config.Port)
  client := &http.Client{}
  request, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return err
  }
  request.Header.Set("Accept", "application/json")
  response, err := client.Do(request)
  if err != nil {
    return err
  }
  body, err := ioutil.ReadAll(response.Body)
  if err != nil {
    return err
  }
  prettyPrintJsonBody(body)
  response.Body.Close()
  return nil
}

func createBadOrder() []order {
  return []order{{
    Name: "error",
    Address: "badhood",
    Items: []item{{5,1}},
  }}
}

func orderItem(config *connection, badRequest bool) error {
  var orders []order
  var err error
  if badRequest {
    orders = createBadOrder()
  } else {
    orders, err = readOrders("./orders")
    if err != nil {
      return err
    }
  }
  url := fmt.Sprintf("http://%s:%d/order", config.Url, config.Port)
  client := &http.Client{}
  for _, order := range orders {
    orderJson, err := json.Marshal(order)
    if err != nil {
      return err
    }

    request, err := http.NewRequest("PUT", url, bytes.NewBuffer(orderJson))
    if err != nil {
      return err
    }
    request.Header.Set("Content-Type", "application/json")
    request.Header.Set("Accept", "application/json")
    response, err := client.Do(request)
    if err != nil {
      return err
    }
    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
      return err
    }
    prettyPrintJsonBody(body)
    response.Body.Close()
  }
  return nil
}

func prettyPrintJsonBody(body []byte) {
  var out bytes.Buffer
  json.Indent(&out, body, "", "  ")
  fmt.Println(out.String())
}

func readOrders(dirName string) ([]order, error) {
  files, err := ioutil.ReadDir(dirName)
  if err != nil {
    return nil, err
  }
  var orders = make([]order, len(files))
  for i, file := range files {
    dat, err := ioutil.ReadFile("./orders/" + file.Name())
    if err != nil {
      fmt.Println("Error while opening: " + file.Name() + "\n" + err.Error())
      continue
    }
    var buff order
    if err := json.Unmarshal(dat, &buff); err != nil {
      fmt.Println("Not a JSON file: " + file.Name())
      continue
    }
    orders[i] = buff
  }
  return orders, nil
}

func configure() (*connection, error) {
  dat, err := ioutil.ReadFile("./config.json")
  if err != nil {
    return nil, err
  }
  var result connection
  if err := json.Unmarshal(dat, &result); err != nil {
    return nil, err
  }
  return &result, nil
}

func main() {
  config, err := configure()
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
  action, err := checkArgs()
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }

  switch action.action {
  case QUERY_ITEMS:
    fmt.Println("Operation: QUERY_ITEMS")
    queryItems(config)
  case QUERY_ITEM_AVAILABILITY:
    fmt.Println("Operation: QUERY_ITEM_AVAILABILITY")
    queryAvailability(config)
  case ORDER_ITEM:
    fmt.Println("Operation: ORDER_ITEM")
    err = orderItem(config, action.badRequest)
  default:
    fmt.Println(getHelp())
    os.Exit(1)
  }
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}

func getHelp() string {
  return fmt.Sprint("Usage: client OPERATION [--error]\n" +
                    "OPERATION\n" +
                    "  * items\n" +
                    "  * availability\n" +
                    "  * order")
}
