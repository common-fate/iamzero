import { Stack } from "@chakra-ui/layout";
import {
  Container,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Text,
} from "@chakra-ui/react";
import React from "react";
import { useHistory } from "react-router-dom";
import { usePolicies } from "../api";
import { Action, Policy } from "../api-types";
import { CenteredSpinner } from "../components/CenteredSpinner";
import { PolicyBox } from "../components/PolicyBox";

export const MOCK_RESOURCES = [
  {
    id: "1",
    name: "iamzero-test-access-bucket",
    description: "S3 Bucket",
    type: "aws:s3:bucket",
  },
];

export const MOCK_ACTIONS: Action[] = [
  {
    id: "d6891731-1d01-4bac-9ed1-35c992d4fd99",
    event: {
      time: "2021-07-14T09:21:41.385841Z",
      data: {
        service: "s3",
        region: "ap-southeast-2",
        operation: "CreateBucket",
        parameters: {
          Account: "123456789",
          Bucket: "iamzero-test-access-bucket",
          CreateBucketConfiguration: {
            LocationConstraint: "ap-southeast-2",
          },
          Region: "ap-southeast-2",
        },
        exceptionMessage: "Access Denied",
        exceptionCode: "AccessDenied",
      },
    },
    enabled: true,
    selectedRecommendation: "934e8218-ddc1-4ae6-a0cd-e86b70f8d96b",
    status: "active",
    time: new Date(Date.parse("2021-07-14T09:21:42.004805954Z")),
    recommendations: [
      {
        ID: "934e8218-ddc1-4ae6-a0cd-e86b70f8d96b",
        Comment: "Allow creating the specific bucket",
        Description: [
          {
            AppliedTo: "iamzero-test-role",
            Type: "IAM Policy",
            Policy: {
              Version: "2012-10-17",
              Id: "934e8218-ddc1-4ae6-a0cd-e86b70f8d96b",
              Statement: [
                {
                  Sid: "iamzero-b2701977-47e3-4eac-a931-80f5948e977f",
                  Effect: "Allow",
                  Action: ["s3:CreateBucket"],
                  Resource: ["arn:aws:s3:::iamzero-test-access-bucket"],
                },
              ],
            },
          },
        ],
      },
      {
        ID: "a6276d02-fb47-4a19-a1f0-4146229fd1c5",
        Comment: "Allow creating all buckets",
        Description: [
          {
            AppliedTo: "iamzero-test-role",
            Type: "IAM Policy",
            Policy: {
              Version: "2012-10-17",
              Id: "a6276d02-fb47-4a19-a1f0-4146229fd1c5",
              Statement: [
                {
                  Sid: "iamzero-9ec53338-819b-4829-b878-818459d94c11",
                  Effect: "Allow",
                  Action: ["s3:CreateBucket"],
                  Resource: ["arn:aws:s3:::*"],
                },
              ],
            },
          },
        ],
      },
    ],
    hasRecommendations: true,
  },
];

export const MOCK_POLICIES: Policy[] = [
  {
    id: "1",
    status: "active",
    identity: {
      account: "123456789",
      user: "iamzero-test",
      role: "arn:aws:iam::123456789:role/iamzero-test-role",
    },
    eventCount: 31,
    lastUpdated: new Date(),
    token: {
      id: "1",
      name: "Chris's laptop",
    },
    document: {
      Version: "2012-10-17",
      Statement: [
        {
          Sid: "1",
          Action: "dynamodb:Query",
          Effect: "Allow",
          Resource: [
            "arn:aws:dynamodb:ap-southeast-2:123456789:table/IAMZero-dev/index/*",
          ],
        },
      ],
    },
  },
  {
    id: "2",
    status: "active",
    identity: {
      account: "123456789",
      user: "second-role",
      role: "arn:aws:iam::123456789:role/iamzero-test-role",
    },
    eventCount: 5,
    lastUpdated: new Date(),
    token: {
      id: "1",
      name: "Chris's laptop",
    },
    document: {
      Version: "2012-10-17",
      Statement: [
        {
          Sid: "1",
          Action: "dynamodb:Query",
          Effect: "Allow",
          Resource: [
            "arn:aws:dynamodb:ap-southeast-2:123456789:table/IAMZero-dev/index/*",
          ],
        },
      ],
    },
  },
];

const Policies: React.FC = () => {
  return (
    <Container maxW="1200px" py={5}>
      <Tabs>
        <TabList>
          <Tab>Active</Tab>
          <Tab>Resolved</Tab>
        </TabList>
        <TabPanels>
          <TabPanel>
            <PolicyList />
          </TabPanel>
          <TabPanel>{/* <FixedAlertList /> */}</TabPanel>
        </TabPanels>
      </Tabs>
    </Container>
  );
};

const PolicyList: React.FC = () => {
  const { data } = usePolicies();
  const history = useHistory();

  if (data === undefined) {
    return <CenteredSpinner />;
  }

  if (data.length === 0)
    return <Text textAlign="center">No policies yet!</Text>;

  return (
    <Stack>
      {data.map((policy) => (
        <PolicyBox
          key={policy.id}
          policy={policy}
          onClick={() => history.push(`/policies/${policy.id}`)}
        />
      ))}
    </Stack>
  );
};

export default Policies;
