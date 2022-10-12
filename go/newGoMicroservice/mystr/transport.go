package mystr

import (
"context"
"encoding/json"
"errors"
"net/http"

"github.com/go-kit/kit/endpoint"
httptransport "github.com/go-kit/kit/transport/http"
)

func GetIsPalHandler(ep endpoint.Endpoint, options []httptransport.ServerOption) *httptransport.Server {
return httptransport.NewServer(
  ep,
  decodeGetIsPalRequest,
  encodeGetIsPalResponse,
  options...,
)
}

func decodeGetIsPalRequest(_ context.Context, r *http.Request) (interface{}, error) {
var req IsPalRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
  return nil, err
}
return req, nil
}

func encodeGetIsPalResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
resp, ok := response.(*IsPalResponse)
if !ok {
  return errors.New("error decoding")
}
return json.NewEncoder(w).Encode(resp)
}

func GetReverseHandler(ep endpoint.Endpoint, options []httptransport.ServerOption) *httptransport.Server {
return httptransport.NewServer(
  ep,
  decodeGetReverseRequest,
  encodeGetReverseResponse,
  options...,
)
}

func decodeGetReverseRequest(_ context.Context, r *http.Request) (interface{}, error) {
var req ReverseRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
  return nil, err
}
return req, nil
}

func encodeGetReverseResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
resp, ok := response.(*ReverseResponse)
if !ok {
  return errors.New("error decoding")
}

return json.NewEncoder(w).Encode(resp)
}
