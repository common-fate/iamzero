import { Center, Link, Text } from "@chakra-ui/react";
import React from "react";
import { Link as RouterLink, Redirect, useParams } from "react-router-dom";
import { useAction } from "../api";
import { CenteredSpinner } from "../components/CenteredSpinner";

const AlertRedirectToPolicy: React.FC = () => {
  const { alertId } = useParams<{ alertId: string }>();
  const { data, error } = useAction(alertId);

  if (error) {
    return (
      <Center flexGrow={1}>
        <Text>
          We couldn't find the action you're looking for.{" "}
          <Link as={RouterLink} to="/policies">
            Click here to go back.
          </Link>
        </Text>
      </Center>
    );
  }

  if (data === undefined) return <CenteredSpinner />;

  return <Redirect to={`/policies/${data.policyId}?action=${data.id}`} />;
};

export default AlertRedirectToPolicy;
