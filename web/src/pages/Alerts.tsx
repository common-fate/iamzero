import { Stack } from "@chakra-ui/layout";
import {
  Button,
  ButtonGroup,
  Container,
  HStack,
  Spinner,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Text,
} from "@chakra-ui/react";
import React, { useEffect, useState } from "react";
import { useParams } from "react-router";
import { reviewAlert, useAlerts } from "../api";
import { Alert } from "../api-types";
import { AlertBox } from "../components/AlertBox";
import { FixAlertModal } from "../components/FixAlertModal";
import { UnhandledAlertModal } from "../components/UnhandledAlertModal";

const Alerts: React.FC = () => {
  return (
    <Tabs>
      <TabList>
        <Tab>Active</Tab>
        <Tab>Fixed</Tab>
        <Tab>Ignored</Tab>
      </TabList>
      <TabPanels>
        <TabPanel>
          <AlertList />
        </TabPanel>
        <TabPanel>
          <FixedAlertList />
        </TabPanel>
        <TabPanel>
          <IgnoredAlertList />
        </TabPanel>
      </TabPanels>
    </Tabs>
  );
};

const AlertList: React.FC = () => {
  const { data: apiData, revalidate } = useAlerts();
  const { alertId } = useParams<{ alertId: string }>();
  const [selectedAlert, setSelectedAlert] = useState<Alert>();

  // If the user visits /alerts/:alertId, show the modal for the
  // specified alert instantly.
  // This allows us to display a user-friendly link in IAM Zero
  // client libraries, where a single click will take the user
  // to the right alert in the IAM Zero console.
  useEffect(() => {
    if (alertId !== undefined && apiData !== undefined) {
      const alertToSelect = apiData.find((a) => a.id === alertId);
      if (alertToSelect !== undefined) {
        setSelectedAlert(alertToSelect);
      }
    }
  }, [alertId, apiData]);

  const onClose = () => setSelectedAlert(undefined);

  const onApplyRecommendation = async (recommendationId: string) => {
    if (selectedAlert !== undefined) {
      await reviewAlert(selectedAlert.id, {
        Decision: "apply",
        RecommendationID: recommendationId,
      });
      setSelectedAlert(undefined);
      void revalidate();
    }
  };

  const onIgnoreRecommendation = async (alertId: string) => {
    await reviewAlert(alertId, {
      Decision: "ignore",
    });
    void revalidate();
  };

  if (apiData === undefined) return <Spinner />;

  const data = apiData.filter(
    (d) => d.status === "active" || d.status === "applying"
  );

  if (data.length === 0) return <Text textAlign="center">No alerts!</Text>;

  return (
    <>
      <Container maxW="1200px" py={5}>
        <Stack>
          {data.map((alert) => (
            <AlertBox key={alert.id} alert={alert}>
              {alert.status === "applying" && (
                <HStack>
                  <Spinner size="xs" />
                  <Text>Applying...</Text>
                </HStack>
              )}
              {alert.status === "active" && (
                <ButtonGroup>
                  <Button onClick={() => onIgnoreRecommendation(alert.id)}>
                    Ignore
                  </Button>
                  <Button
                    onClick={() => setSelectedAlert(alert)}
                    colorScheme="blue"
                  >
                    Fix
                  </Button>
                </ButtonGroup>
              )}
            </AlertBox>
          ))}
        </Stack>
      </Container>
      {/* Modal for alerts which we have a policy mapping for */}
      {selectedAlert && selectedAlert.hasRecommendations && (
        <FixAlertModal
          alert={selectedAlert}
          onClose={onClose}
          onApplyRecommendation={onApplyRecommendation}
        />
      )}
      {/* Modal for alerts which no mapping exists for yet */}
      {selectedAlert && !selectedAlert.hasRecommendations && (
        <UnhandledAlertModal alert={selectedAlert} onClose={onClose} />
      )}
    </>
  );
};

const FixedAlertList: React.FC = () => {
  const { data: apiData } = useAlerts();
  if (apiData === undefined) return <Spinner />;
  const data = apiData.filter((d) => d.status === "fixed");

  if (data.length === 0) return <Text textAlign="center">No alerts!</Text>;

  return (
    <Stack>
      {data.map((alert) => (
        <AlertBox key={alert.id} alert={alert} />
      ))}
    </Stack>
  );
};

const IgnoredAlertList: React.FC = () => {
  const { data: apiData } = useAlerts();
  if (apiData === undefined) return <Spinner />;
  const data = apiData.filter((d) => d.status === "ignored");

  if (data.length === 0) return <Text textAlign="center">No alerts!</Text>;

  return (
    <Stack>
      {data.map((alert) => (
        <AlertBox key={alert.id} alert={alert} />
      ))}
    </Stack>
  );
};

export default Alerts;
