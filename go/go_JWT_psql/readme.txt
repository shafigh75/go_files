sample signup :
curl -XPOST  localhost:8080/signup -d '{"email":"test@test.com","password":"102030", "role":"admin"}'

sample sign-in:
curl -XPOST  localhost:8080/signin -d '{"email":"test@test.com","password":"102030"}'

sample access admin panel:
curl -XGET localhost:8080/admin --header 'Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhdXRob3JpemVkIjp0cnVlLCJlbWFpbCI
6InRlc3RAdGVzdC5jb20iLCJleHAiOjE2NjA5OTI1NTAsInJvbGUiOiJhZG1pbiJ9.dphFv8vjR9kXkN7N2KHuKJ7BfPn2uUbgcZdjOfLGh5Q'

URL for text tutorial:
https://www.bacancytechnology.com/blog/golang-jwt
