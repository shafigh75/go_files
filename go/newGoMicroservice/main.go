package main

import (
"net/http"
"os"

"m/mystr"
"github.com/go-kit/kit/endpoint"
kitlog "github.com/go-kit/kit/log"
"github.com/go-kit/kit/log/level"
httptransport "github.com/go-kit/kit/transport/http"
"github.com/gorilla/mux"
)

func main() {
var logger kitlog.Logger
{
  logger = kitlog.NewLogfmtLogger(os.Stderr)
  logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
  logger = kitlog.With(logger, "caller", kitlog.DefaultCaller)
}

var middlewares []endpoint.Middleware
var options []httptransport.ServerOption
svc := mystr.NewService()
eps := mystr.MakeEndpoints(svc, logger, middlewares)
r := mux.NewRouter()
r.Methods(http.MethodGet).Path("/palindrome").Handler(mystr.GetIsPalHandler(eps.GetIsPalindrome, options))
r.Methods(http.MethodGet).Path("/reverse").Handler(mystr.GetReverseHandler(eps.GetReverse, options))
level.Info(logger).Log("status", "listening", "port", "8080")
svr := http.Server{
  Addr:    "127.0.0.1:8080",
  Handler: r,
}
level.Error(logger).Log(svr.ListenAndServe())
}
