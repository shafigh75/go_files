Below is a comprehensive yet concise MongoDB tutorial covering installation, data modeling, CRUD operations, querying, indexing, aggregation, and more. This guide focuses on the most important commands and concepts for programmers.

---

# MongoDB Full Tutorial

## 1. What is MongoDB?

MongoDB is a NoSQL, document-oriented database designed for scalability and high performance. Data is stored in JSON-like documents (BSON) which makes it flexible and schema-less.

---

## 2. Installation and Setup

### a. Installing MongoDB

- **On macOS using Homebrew:**

  ```bash
  brew tap mongodb/brew
  brew install mongodb-community@6.0
  ```

- **On Ubuntu:**

  Follow the official guide:
  
  ```bash
  wget -qO - https://www.mongodb.org/static/pgp/server-6.0.asc | sudo apt-key add -
  echo "deb [ arch=amd64,arm64 ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/6.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-6.0.list
  sudo apt-get update
  sudo apt-get install -y mongodb-org
  ```

- **On Windows:**  
  Download the MSI installer from MongoDB’s official website and follow the installation wizard.

### b. Running the Server

Start the MongoDB daemon (mongod):

```bash
mongod
```

In a separate terminal, launch the MongoDB shell:

```bash
mongo
```

---

## 3. Basic Terminology

- **Database:** A container for collections.
- **Collection:** A group of MongoDB documents (similar to a table in relational databases).
- **Document:** A set of key-value pairs within a collection (similar to a row in relational databases).

---

## 4. CRUD Operations

### a. Create

#### Insert a Single Document

```javascript
db.collectionName.insertOne({ name: "Alice", age: 30, city: "New York" });
```

#### Insert Multiple Documents

```javascript
db.collectionName.insertMany([
  { name: "Bob", age: 25, city: "Los Angeles" },
  { name: "Charlie", age: 35, city: "Chicago" }
]);
```

### b. Read (Query)

#### Find All Documents

```javascript
db.collectionName.find();
```

#### Find with Query Conditions

```javascript
db.collectionName.find({ age: { $gt: 28 } });
```

#### Find One Document

```javascript
db.collectionName.findOne({ name: "Alice" });
```

### c. Update

#### Update a Single Document

```javascript
db.collectionName.updateOne(
  { name: "Alice" }, 
  { $set: { age: 31, city: "Boston" } }
);
```

#### Update Multiple Documents

```javascript
db.collectionName.updateMany(
  { city: "Chicago" },
  { $set: { city: "Houston" } }
);
```

#### Replace a Document

```javascript
db.collectionName.replaceOne(
  { name: "Alice" },
  { name: "Alice", age: 31, city: "Boston", phone: "123-4567" }
);
```

### d. Delete

#### Delete a Single Document

```javascript
db.collectionName.deleteOne({ name: "Bob" });
```

#### Delete Multiple Documents

```javascript
db.collectionName.deleteMany({ city: "Houston" });
```

---

## 5. Query Operators & Projections

### a. Query Operators

- **Comparison Operators**

  - `$gt` (greater than), `$gte` (greater than or equal)
  - `$lt` (less than), `$lte` (less than or equal)
  - `$ne` (not equal)

  ```javascript
  db.collectionName.find({ age: { $gte: 30 } });
  ```

- **Logical Operators**

  - `$and`, `$or`, `$not`
  
  ```javascript
  db.collectionName.find({ $or: [{ city: "Boston" }, { age: { $lt: 28 } }] });
  ```

- **Element Query Operators**

  - `$exists`
  
  ```javascript
  db.collectionName.find({ phone: { $exists: true } });
  ```

### b. Projections

Limit the fields returned in the query.

```javascript
db.collectionName.find({ age: { $gt: 25 } }, { name: 1, _id: 0 });
```

---

## 6. Indexing

Indexes improve query performance.

### a. Creating an Index

```javascript
db.collectionName.createIndex({ name: 1 });
```

### b. Dropping an Index

```javascript
db.collectionName.dropIndex("name_1");
```

### c. List All Indexes

```javascript
db.collectionName.getIndexes();
```

---

## 7. Aggregation Framework

The aggregation framework processes data records and returns computed results.

### a. Basic Pipeline Example

```javascript
db.collectionName.aggregate([
  { $match: { age: { $gte: 30 } } },
  { $group: { _id: "$city", averageAge: { $avg: "$age" } } },
  { $sort: { averageAge: -1 } }
]);
```

### b. Common Stages

- `$match`: Filters the documents.
- `$group`: Groups documents with the same key.
- `$sort`: Sorts documents.
- `$project`: Reshapes each document.
- `$limit`/`$skip`: Controls the number of documents passed.

---

## 8. Data Modeling Tips

- Use embedded documents to model one-to-few relationships.
- Use references (manual associations) for one-to-many or many-to-many relationships.
- Design schemas based on your query patterns and performance requirements.

---

## 9. MongoDB Shell & Drivers

### a. MongoDB Shell (mongosh)

The new MongoDB shell (mongosh) supports modern JavaScript practices and provides an improved interactive experience.

```bash
mongosh
```

### b. Drivers

MongoDB provides drivers for several programming languages (Node.js, Python, Java, etc.). For example, a basic Node.js connection:

```javascript
const { MongoClient } = require("mongodb");

async function main() {
  const uri = "mongodb://localhost:27017";
  const client = new MongoClient(uri);
  
  try {
    await client.connect();
    const database = client.db("myDatabase");
    const collection = database.collection("myCollection");
    
    const result = await collection.insertOne({ name: "Alice", age: 30 });
    console.log("Inserted document id:", result.insertedId);
  } finally {
    await client.close();
  }
}

main().catch(console.error);
```

---

## 10. Backup & Replication Basics

### a. Creating a Backup

Use `mongodump` to back up a database:

```bash
mongodump --db myDatabase --out /backup/dir
```

### b. Restoring a Backup

Use `mongorestore` to restore:

```bash
mongorestore --db myDatabase /backup/dir/myDatabase
```

### c. Replication

MongoDB supports replica sets for high availability. A replica set consists of multiple mongod processes that maintain the same data set.

- Initialize a replica set on the primary:

  ```javascript
  rs.initiate({
    _id: "rs0",
    members: [
      { _id: 0, host: "localhost:27017" },
      { _id: 1, host: "localhost:27018" },
      { _id: 2, host: "localhost:27019" }
    ]
  });
  ```

---

## 11. Sharding

Sharding partitions data across multiple servers. Key steps include:

1. Enable sharding for a database:
   ```javascript
   sh.enableSharding("myDatabase");
   ```

2. Shard a collection based on a key:
   ```javascript
   sh.shardCollection("myDatabase.myCollection", { shardKey: 1 });
   ```

Sharding helps with horizontal scaling.

---

## 12. Useful Administrative Commands

- **Show Databases/Collections:**

  ```javascript
  show dbs;
  use myDatabase;
  show collections;
  ```

- **Database Stats:**

  ```javascript
  db.stats();
  db.collectionName.stats();
  ```

- **Current Operations:**

  ```javascript
  db.currentOp();
  ```

---

## 13. Security Best Practices

- Enable authentication and create admin users.
- Use role-based access control (RBAC).
- Configure network settings (bind IPs) and firewall rules.
- Use TLS/SSL for data encryption in transit.

Example: Create an admin user

```javascript
use admin;
db.createUser({
  user: "admin",
  pwd: "securePassword",
  roles: [{ role: "root", db: "admin" }]
});
```

Enable authentication in the MongoDB config file (`mongod.conf`):

```yaml
security:
  authorization: "enabled"
```

---

## 14. Advanced Topics & Resources

- Replica Set and Sharding configurations in production.
- Monitoring with MongoDB Cloud Manager or third-party tools.
- Understanding write concerns and read preferences.
- Official MongoDB documentation: https://docs.mongodb.com/
- Community forums and courses for deeper dives.

---

# Conclusion

This tutorial provided a concise yet comprehensive overview of MongoDB and its essential commands and features—from CRUD operations and indexing to aggregation and security practices. Experiment with these commands in a local environment and explore production-level configurations as you progress. Happy coding!
