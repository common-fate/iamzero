import { Flex, Heading, Stack, Text } from "@chakra-ui/layout";
import React from "react";
import { format } from "timeago.js";
import { Policy } from "../api-types";
import { FaUserAlt } from "react-icons/fa";
import { Badge, Box, Icon } from "@chakra-ui/react";
import { getEventCountString } from "../utils/getEventCountString";
import { RelativeDateText } from "./LastUpdatedText";

interface Props {
  policy: Policy;
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
        <Text>
          <Icon as={FaUserAlt} w="12px" /> {policy.token.name}
        </Text>
      </Stack>
      <Flex direction="column" justify="space-between" align="flex-end">
        <RelativeDateText textAlign="right" date={policy.lastUpdated} />
        <Box>
          <Badge>{getEventCountString(policy.eventCount)}</Badge>
        </Box>
      </Flex>
    </Flex>
  );
};
