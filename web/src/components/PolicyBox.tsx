import { Flex, Heading, Stack, Text } from "@chakra-ui/layout";
import React from "react";
import { format } from "timeago.js";
import { Policy } from "../api-types";
import { FaUserAlt } from "react-icons/fa";
import { Badge, Box, Icon } from "@chakra-ui/react";

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
      <Stack>
        <Heading size="sm" textAlign="left">
          {policy.identity.role}
        </Heading>
        <Text>
          <Icon as={FaUserAlt} w="12px" /> {policy.token.name}
        </Text>
      </Stack>
      <Flex direction="column" justify="space-between" align="flex-end">
        <Text textAlign="right">{format(policy.lastUpdated)}</Text>
        <Box>
          <Badge>{policy.eventCount} events</Badge>
        </Box>
      </Flex>
    </Flex>
  );
};
