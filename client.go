/*
Copyright (c) 2017 Josef Karasek

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
 */
package main

import (
  "os"
  "fmt"
  "errors"
  "bytes"
  "net/http"
  "encoding/json"
  "io/ioutil"
  "sync"
  "time"
  "strconv"
)

type item struct {
  CatalogId int  `json:"catalogId"`
  Amount    int     `json:"amount"`
}

type order struct {
  Name    string     `json:"name"`
  Address string  `json:"address"`
  Items   []item    `json:"items"`
}

type action struct {
  action     int
  id         int
  badRequest bool
}

type connection struct {
  Url  string `json:"url"`
  Port int   `json:"port"`
}

const (
  QUERY_ITEMS             = iota
  QUERY_ITEM_AVAILABILITY
  ORDER_ITEM
)

var wg sync.WaitGroup

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
    return &action{parseOperation(os.Args[1]), -1, false}, nil
  case 3:
    if os.Args[2] == "--error" {
      return &action{parseOperation(os.Args[1]), -1, true}, nil
    }
    if val, err := strconv.Atoi(os.Args[2]); err == nil {
      return &action{parseOperation(os.Args[1]), val, false}, nil
    }
  case 4:
    if val, err := strconv.Atoi(os.Args[2]); err == nil && os.Args[3] == "--error" {
      return &action{parseOperation(os.Args[1]), val, true}, nil
    }
  }
  return nil, errors.New(getHelp())
}

func queryItems(config *connection, isId int) error {
  if isId == -1 {
    return queryAllCatalogs(config)
  } else {
    return queryCatalog(config, isId)
  }
}

func queryCatalog(config *connection, id int) error {
  client := &http.Client{}
  for true {
    start := time.Now()
    url := fmt.Sprintf("http://%s:%d/catalog/list/%d", config.Url, config.Port, id)
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
    elapsed := time.Since(start)
    fmt.Printf("Time elapsed: %s | Response: %s\n", elapsed.String(), string(body))
    response.Body.Close()

    time.Sleep(2000 * time.Millisecond)
  }
  return nil
}

func queryAllCatalogs(config *connection) error {
  client := &http.Client{}
  for true {
    for category, _ := range []int{0, 1, 2} {
      start := time.Now()
      url := fmt.Sprintf("http://%s:%d/catalog/list/%d", config.Url, config.Port, category)
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
      elapsed := time.Since(start)
      fmt.Printf("Time elapsed: %s | Response: %s\n", elapsed.String(), string(body))
      response.Body.Close()
    }
    time.Sleep(2000 * time.Millisecond)
  }
  return nil
}

func getId(isId, _default int) int {
  if isId == -1 {
    return _default
  } else {
    return isId
  }
}

func queryAvailability(config *connection, isId int) error {
  url := fmt.Sprintf("http://%s:%d/availability/%d", config.Url, config.Port, getId(isId, 5))
  client := &http.Client{}
  request, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return err
  }
  request.Header.Set("Accept", "application/json")

  for true {
    start := time.Now()
    response, err := client.Do(request)
    if err != nil {
      return err
    }
    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
      return err
    }
    elapsed := time.Since(start)
    fmt.Printf("Time elapsed: %s | Response: %s\n", elapsed.String(), string(body))
    response.Body.Close()

    time.Sleep(2000 * time.Millisecond)
  }
  return nil
}

func createBadOrder() *order {
  return &order{
    Name:    "error",
    Address: "badhood",
    Items:   []item{{5, 1}},
  }
}

func orderItem(config *connection, badRequest bool) error {
  var newOrder *order
  var err error
  if badRequest {
    newOrder = createBadOrder()
  } else {
    newOrder, err = readOrders("./orders")
    if err != nil {
      return err
    }
  }
  url := fmt.Sprintf("http://%s:%d/order", config.Url, config.Port)
  client := &http.Client{}

  // Server side has limit of 20 connections
  // If we wanna block the server we need to block all of those threads
  var connections int
  if badRequest {
    connections = 20
  } else {
    connections = 1
  }

  orderJson, err := json.Marshal(newOrder)
  if err != nil {
    return err
  }

  for true {
    wg.Add(connections)
    for i := 0; i < connections; i++ {
      go func(id int) {
        start := time.Now()
        request, err := http.NewRequest("PUT", url, bytes.NewBuffer(orderJson))
        if err != nil {
          fmt.Printf("%d: Error %s\n", id, err)
          return
        }
        request.Header.Set("Content-Type", "application/json")
        request.Header.Set("Accept", "application/json")
        response, err := client.Do(request)
        if err != nil {
          fmt.Printf("%d: Error %s\n", id, err)
          return
        }
        body, err := ioutil.ReadAll(response.Body)
        if err != nil {
          fmt.Printf("%d: Error %s\n", id, err)
          return
        }
        elapsed := time.Since(start)
        fmt.Printf("Time elapsed: %s | Response: %s\n", elapsed.String(), string(body))
        response.Body.Close()
        wg.Done()
      }(i)
    }
    wg.Wait()
    time.Sleep(2000 * time.Millisecond)
  }
  return nil
}

func prettyPrintBody(body []byte) {
  var out bytes.Buffer
  // check if body is json
  err := json.Indent(&out, body, "", "  ")
  if err != nil {
    fmt.Println(string(body))
  } else {
    fmt.Println(out.String())
  }
}

func readOrders(dirName string) (*order, error) {
  files, err := ioutil.ReadDir(dirName)
  if err != nil {
    return nil, err
  }
  if len(files) == 0 {
    return nil, errors.New("No files found in ./orders")
  }
  dat, err := ioutil.ReadFile("./orders/" + files[0].Name())
  if err != nil {
    return nil, err
  }
  var buff order
  if err := json.Unmarshal(dat, &buff); err != nil {
    return nil, err
  }
  return &buff, nil
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
    err = queryItems(config, action.id)
  case QUERY_ITEM_AVAILABILITY:
    fmt.Println("Operation: QUERY_ITEM_AVAILABILITY")
    err = queryAvailability(config, action.id)
  case ORDER_ITEM:
    fmt.Println("Operation: ORDER_ITEM")
    err = orderItem(config, action.badRequest)
  default:
    err = errors.New(getHelp())
  }
  if err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}

func getHelp() string {
  return fmt.Sprint("Usage: client OPERATION [item_id] [--error]\n" +
    "\n" +
    "OPERATION\n" +
    "  * items\n" +
    "  * availability\n" +
    "  * order\n" +
    "\n" +
    "Examples:\n" +
    "  # List items in all categories\n" +
    "  client items\n" +
    "\n" +
    "  # List items in category 0 (available categories are 0,1,2)\n" +
    "  client items 0\n" +
    "\n" +
    "  # Query availability of item with id 1\n" +
    "  client availability 1\n" +
    "\n" +
    "  # Query availability of an arbitrary item\n" +
    "  client availability\n" +
    "\n" +
    "  # Send order from 'orders' dir to server\n" +
    "  client order\n" +
    "\n" +
    "  # Send order to a slow endpoint\n" +
    "  client order --error")
}
