# Contributing to iamzero

We welcome all contributions to IAM Zero. Please read our [Contributor Code of Conduct](./CODE_OF_CONDUCT.md).

## Getting set up

This project is a web service with the backend written in Go and frontend written in React and TypeScript. To run this project locally you will need:

- Go 1.16
- NodeJS v14
- Yarn 1.22

This project utilises Go Modules to manage Go dependencies.

The frontend is in the `web` folder in the repo. To build and run the frontend, first install the NodeJS dependencies:

```
cd web
yarn install
```

Then run the React application:

```
yarn start
```

You will see the app on http://localhost:3000 by default.

To build and run the backend, run the command:

```
go run cmd/main.go
```

The backend API is served on http://localhost:9090 by default.

# CloudFormation templates

CloudFormation templates are currently a work-in-progress. These can be packaged and deployed as follows (note: requires access to the iamzero sandbox AWS account, otherwise you can create your own S3 bucket):

```
aws cloudformation package --template-file ./deploy/root.yml --output-template ./deploy/packaged.yml --s3-bucket iamzero-dev-cloudformation

aws cloudformation deploy --template-file TEMPLATE_FILE_FROM_PREVIOUS_STEP --stack-name iamzero --parameter-overrides CertificateArn=<CERTIFICATE_ARN_FOR_IAMZERO>
```
