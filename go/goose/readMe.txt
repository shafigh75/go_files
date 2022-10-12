goose is for database migrations.

for downloading and installing it i had to use tor as proxy :
1. open tor ( it opens a socks5 proxy on 9150 )
2. use the export https_proxy="socks5://127.0.0.1:9150" 
3. $ go install github.com/pressly/goose/v3/cmd/goose@latest

____________________________________________________________________

now that it is installed we can use it via commandline tools:

1. check db Status:
goose mysql "mohammad:9090kosenanat@/test?parseTime=true" status

2. create a go/sql mock sql migration file:
goose mysql "mohammad:9090kosenanat@/test?parseTime=true" create <FILE_NAME> <sql|go>

3. fill the sql file like this:
-- +goose Up
CREATE TABLE post (
    id int NOT NULL,
    title text,
    body text,
    PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE post;


4. up (apply) migration :
goose mysql "mohammad:9090kosenanat@/test?parseTime=true" up 

5. down (tearDown) :
goose mysql "mohammad:9090kosenanat@/test?parseTime=true" down 




