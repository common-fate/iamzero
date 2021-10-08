import { Flex, Heading, Stack } from "@chakra-ui/layout";
import { Badge, Box } from "@chakra-ui/react";
import React from "react";
import { Finding } from "../api-types";
import { getEventCountString } from "../utils/getEventCountString";
import { RelativeDateText } from "./LastUpdatedText";

interface Props {
  policy: Finding;
  onClick?: () => void;
}

export const PolicyBox: React.FC<Props> = ({ policy, onClick }) => {
  return (
    <Flex
      onClick={onClick}
      as="button"
      bg="white"
      p={3}
      borderRadius={5}
      shadow="sm"
      flexDir="row"
      justify="space-between"
      borderColor="gray.300"
      borderWidth="thin"
    >
      <Stack align="flex-start">
        <Heading size="sm" textAlign="left">
          {policy.identity.role}
        </Heading>
      </Stack>
      <Flex direction="column" justify="space-between" align="flex-end">
        <RelativeDateText textAlign="right" date={policy.updatedAt} />
        <Box>
          <Badge>{getEventCountString(policy.eventCount)}</Badge>
        </Box>
      </Flex>
    </Flex>
  );
};
