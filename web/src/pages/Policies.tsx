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
import { PolicyStatus } from "../api-types";
import { CenteredSpinner } from "../components/CenteredSpinner";
import { PolicyBox } from "../components/PolicyBox";

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
            <PolicyList status="active" />
          </TabPanel>
          <TabPanel>
            <PolicyList status="resolved" />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </Container>
  );
};

interface PolicyListProps {
  status: PolicyStatus;
}

const PolicyList: React.FC<PolicyListProps> = ({ status }) => {
  const { data } = usePolicies(status);
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
