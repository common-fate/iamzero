import { Avatar } from "@chakra-ui/avatar";
import { Flex, Heading, HStack, Stack, Text } from "@chakra-ui/layout";
import { Box } from "@chakra-ui/react";
import React from "react";
import { format } from "timeago.js";
import { Alert } from "../api-types";
import { getAlertTitle } from "../utils/getAlertTitle";
import { renderStringOrObject } from "../utils/renderStringOrObject";

interface Props {
  alert: Alert;
}

export const AlertBox: React.FC<Props> = ({ alert, children }) => {
  return (
    <Flex
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
        <Heading size="sm">{getAlertTitle(alert)}</Heading>
        <HStack>
          {alert.event.data.parameters &&
            Object.entries(alert.event.data.parameters).map(([key, value]) => (
              <Box as="span" key={key} display="inline-block">
                {key}: <b>{renderStringOrObject(value)}</b>
              </Box>
            ))}
        </HStack>
        <HStack>
          <Avatar size="xs" />
          {/* <Text>{alert.event.identity.user}</Text> */}
        </HStack>
      </Stack>
      <Flex direction="column" justify="space-between">
        <Text textAlign="right">{format(alert.time)}</Text>
        {children}
      </Flex>
    </Flex>
  );
};
