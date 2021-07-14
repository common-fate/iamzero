# Token metadata storage

IAM Zero allows users to create tokens for developers or services. These tokens allow an IAM Zero client to send IAM events to the IAM Zero server.

We implement a Golang interface called `TokenStorer` for token storage. Any storage driver (e.g. a database or cache like Postgres, Redis, DynamoDB) can implement this interface, so that we have some flexibility.

Our initial implementation uses DynamoDB. We will list some operational requirements for DynamoDB below; eventually these will be pushed into the main IAM Zero documentation and our reference deployment architecture.

## DynamoDB token storage

The DynamoDB table must have a primary key called `id`.
