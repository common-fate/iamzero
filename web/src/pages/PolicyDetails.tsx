import { InfoOutlineIcon } from "@chakra-ui/icons";
import {
  Badge,
  Box,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  Button,
  Checkbox,
  Code,
  Flex,
  Heading,
  HStack,
  IconButton,
  Select,
  Stack,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useClipboard,
} from "@chakra-ui/react";
import React, { useState } from "react";
import { Link as RouterLink, useParams } from "react-router-dom";
import { useActionsForPolicy, usePolicy } from "../api";
import { Action } from "../api-types";
import { CenteredSpinner } from "../components/CenteredSpinner";
import { KeyValueBadge } from "../components/KeyValueBadge";
import { RelativeDateText } from "../components/LastUpdatedText";
import { S3Icon } from "../icons";
import { getAlertTitle } from "../utils/getAlertTitle";
import { getEventCountString } from "../utils/getEventCountString";
import { renderStringOrObject } from "../utils/renderStringOrObject";
import { MOCK_RESOURCES } from "./Policies";

const PolicyDetails: React.FC = () => {
  const { policyId } = useParams<{ policyId: string }>();
  const { data: policy } = usePolicy(policyId);
  const { data: actions } = useActionsForPolicy(policyId);
  const { hasCopied, onCopy } = useClipboard(JSON.stringify(policy, null, 2));

  const [selectedStatement, setSelectedStatement] = useState<string>();

  if (policy === undefined || actions === undefined) return <CenteredSpinner />;

  return (
    <Flex flexGrow={1}>
      <Stack flexGrow={1} p={5}>
        <Breadcrumb>
          <BreadcrumbItem>
            <BreadcrumbLink as={RouterLink} to="/policies">
              Policies
            </BreadcrumbLink>
          </BreadcrumbItem>

          <BreadcrumbItem isCurrentPage>
            <BreadcrumbLink href="#">Details</BreadcrumbLink>
          </BreadcrumbItem>
        </Breadcrumb>
        <Stack
          bg="white"
          p={3}
          borderRadius={5}
          shadow="sm"
          justify="space-between"
          borderColor="gray.300"
          borderWidth="thin"
          spacing={8}
        >
          <Stack spacing={4}>
            <Flex direction="row" justify="space-between" spacing={5}>
              <Heading size="md">{policy.identity.role}</Heading>
              <HStack align="flex-end" spacing={5}>
                <Box>
                  <Badge>{getEventCountString(policy.eventCount)}</Badge>
                </Box>
                <RelativeDateText textAlign="right" date={policy.lastUpdated} />
              </HStack>
            </Flex>
            <Stack direction="row" wrap="wrap" spacing={3}>
              <KeyValueBadge label="Role ARN" value={policy.identity.role} />
              <KeyValueBadge label="Account" value={policy.identity.account} />
              <KeyValueBadge label="Token" value={policy.token.name} />
            </Stack>
            <Text>
              The actions below have been recorded by IAM Zero for this role.
              Confirm your IAM policy by selecting actions and then clicking the
              Copy button on the generated policy.
            </Text>
          </Stack>
          <Stack>
            <Flex
              px={3}
              color="gray.500"
              fontSize="xs"
              fontWeight="semibold"
              textAlign="center"
              textTransform="uppercase"
              flexGrow={1}
            >
              <Flex w="100px" justify="center">
                <Text>Enabled</Text>
              </Flex>
              <Flex w="200px" justify="flex-end">
                <Text>Action</Text>
              </Flex>
              <Flex w="350px" justify="flex-end">
                <Text>Resources</Text>
              </Flex>
              <Flex justify="flex-end" flexGrow={1} mr="50px">
                <Text>Advisory</Text>
              </Flex>
            </Flex>
            {actions.map((action) => (
              <ActionDisplay
                key={action.id}
                action={action}
                onMouseOver={() => setSelectedStatement("1")}
                onMouseOut={() => setSelectedStatement(undefined)}
              />
            ))}
          </Stack>
        </Stack>
      </Stack>
      <Flex
        direction="column"
        py={3}
        position="relative"
        backgroundColor="#011627"
      >
        <Box
          width="full"
          mb="1"
          userSelect="none"
          zIndex="0"
          letterSpacing="wide"
          color="gray.400"
          fontSize="xs"
          fontWeight="semibold"
          textAlign="center"
          textTransform="uppercase"
          pointerEvents="none"
        >
          Generated Policy
        </Box>
        <Flex flexGrow={1}>
          {/* NB: We deconstruct the JSON so that we can highlight the corresponding
        IAM policy statements when the user selects an action. However this requires 
        a lot of manual styling rather than just dumping the JSON object. There might be
        a more elegant way to achieve this.
         */}
          <Code
            backgroundColor="#011627"
            display="block"
            whiteSpace="pre"
            px={0}
            color="white"
          >
            <Box as="span" display="block" px={5}>{`{`}</Box>
            <Box as="span" display="block" px={5}>
              {"  "}"Version": "{policy.document.Version}",
            </Box>
            <Box as="span" display="block" px={5}>
              {"  "}"Statement": [
            </Box>
            {policy.document.Statement.map((statement) => (
              <Box
                position="relative"
                as="span"
                display="block"
                backgroundColor={
                  statement.Sid === selectedStatement
                    ? "whiteAlpha.200"
                    : undefined
                }
                px={5}
              >
                {statement.Sid === selectedStatement && (
                  <Text
                    fontFamily="body"
                    position="absolute"
                    top={2}
                    right={2}
                    color="whiteAlpha.700"
                    fontWeight="bold"
                  >
                    s3:CreateBucket
                  </Text>
                )}
                {"    " +
                  JSON.stringify(statement, null, 2).replace(/\n/g, "\n    ")}
              </Box>
            ))}
            <Box as="span" display="block" px={5}>
              {"  "}]
            </Box>
            <Box as="span" display="block" px={5}>
              {`}`}
            </Box>
          </Code>
        </Flex>
        <Flex justify="center" p={5}>
          <Button colorScheme="teal" zIndex="1" onClick={onCopy}>
            {hasCopied ? "Policy Copied!" : "Copy Policy"}
          </Button>
        </Flex>
      </Flex>
    </Flex>
  );
};

interface ActionDisplayProps {
  action: Action;
  onMouseOver?: () => void;
  onMouseOut?: () => void;
}

const ActionDisplay: React.FC<ActionDisplayProps> = ({
  action,
  onMouseOut,
  onMouseOver,
}) => {
  const [expanded, setExpanded] = useState(false);

  return (
    <Stack
      key={action.id}
      borderWidth="1px"
      p={3}
      boxSizing="border-box"
      borderRadius={5}
      onMouseOver={onMouseOver}
      onMouseOut={onMouseOut}
      _hover={{ borderColor: "transparent", boxShadow: "0 0 0 2px #90CDF4" }}
    >
      <Flex align="center" justify="space-between" flexGrow={1}>
        <Flex align="center">
          <Flex w="100px" justify="center">
            <Checkbox defaultChecked />
          </Flex>
          <Flex w="200px" justify="flex-end">
            <Text fontWeight="bold">{getAlertTitle(action)}</Text>
          </Flex>
          <Flex w="350px" justify="flex-end">
            <Box
              as="span"
              borderRadius={5}
              borderWidth="2px"
              p={1}
              backgroundColor="gray.100"
            >
              <S3Icon h="24px" mr={1} />

              {MOCK_RESOURCES[0].name}
            </Box>
          </Flex>
        </Flex>
        <HStack>
          <Select defaultValue={0} maxW="400px">
            {action.recommendations?.map((policy) => (
              <option key={policy.ID} value={policy.ID}>
                {policy.Comment}
              </option>
            ))}
          </Select>
          <IconButton
            variant="ghost"
            icon={<InfoOutlineIcon />}
            onClick={() => setExpanded(!expanded)}
            aria-label="edit"
          />
        </HStack>
      </Flex>
      {expanded && (
        <Stack>
          <Table size="sm">
            <Thead>
              <Tr>
                <Th>Parameter</Th>
                <Th>Value</Th>
              </Tr>
            </Thead>
            <Tbody>
              {Object.entries(action.event.data.parameters).map(
                ([key, value]) => (
                  <Tr key={key}>
                    <Td>{key}</Td>
                    <Td>{renderStringOrObject(value)}</Td>
                  </Tr>
                )
              )}
            </Tbody>
          </Table>
        </Stack>
      )}
    </Stack>
  );
};

export default PolicyDetails;
