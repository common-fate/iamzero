import { Center, Link, Text } from "@chakra-ui/react";
import React from "react";
import { Link as RouterLink, Redirect, useParams } from "react-router-dom";
import { useAction } from "../api";
import { CenteredSpinner } from "../components/CenteredSpinner";

const AlertRedirectToFinding: React.FC = () => {
  const { alertId } = useParams<{ alertId: string }>();
  const { data, error } = useAction(alertId);

  if (error) {
    return (
      <Center flexGrow={1}>
        <Text>
          We couldn't find the action you're looking for.{" "}
          <Link as={RouterLink} to="/findings">
            Click here to go back.
          </Link>
        </Text>
      </Center>
    );
  }

  if (data === undefined) return <CenteredSpinner />;

  return <Redirect to={`/findings/${data.findingId}?action=${data.id}`} />;
};

export default AlertRedirectToFinding;
