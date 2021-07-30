import { CheckIcon, InfoOutlineIcon } from "@chakra-ui/icons";
import {
  Badge,
  Box,
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  Button,
  Center,
  Checkbox,
  Code,
  Flex,
  Heading,
  HStack,
  IconButton,
  Link,
  Select,
  Spinner,
  Stack,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tooltip,
  Tr,
  useClipboard,
  useRadio,
  useRadioGroup,
  UseRadioProps,
} from "@chakra-ui/react";
import produce from "immer";
import React, { useState } from "react";
import { Link as RouterLink, useParams } from "react-router-dom";
import { StringParam, useQueryParam } from "use-query-params";
import {
  editAction,
  EditActionRequestBody,
  setPolicyStatus,
  useActionsForPolicy,
  usePolicy,
} from "../api";
import { Action, PolicyStatus } from "../api-types";
import { CenteredSpinner } from "../components/CenteredSpinner";
import { KeyValueBadge } from "../components/KeyValueBadge";
import { RelativeDateText } from "../components/LastUpdatedText";
import { S3Icon } from "../icons";
import { getAlertTitle } from "../utils/getAlertTitle";
import { getEventCountString } from "../utils/getEventCountString";
import { MOCK_RESOURCES } from "../utils/mockData";
import { renderStringOrObject } from "../utils/renderStringOrObject";

const PolicyDetails: React.FC = () => {
  const { policyId } = useParams<{ policyId: string }>();

  const { data: policy, mutate, error } = usePolicy(policyId);
  const { data: actions, mutate: mutateActions } = useActionsForPolicy(
    policyId
  );
  const [loadingPolicy, setLoadingPolicy] = useState(false);
  const { hasCopied, onCopy } = useClipboard(JSON.stringify(policy, null, 2));

  const [selectedActionId, setSelectedActionId] = useQueryParam(
    "action",
    StringParam
  );

  const { getRootProps, getRadioProps } = useRadioGroup({
    name: "actions",
    value: selectedActionId ?? undefined,
    onChange: (value) => setSelectedActionId(value),
  });

  const group = getRootProps();

  if (error) {
    return (
      <Center flexGrow={1}>
        <Text>
          We couldn't find the policy you're looking for.{" "}
          <Link as={RouterLink} to="/policies">
            Click here to go back.
          </Link>
        </Text>
      </Center>
    );
  }

  if (policy === undefined || actions === undefined) return <CenteredSpinner />;

  const onSetPolicyStatus = async (status: PolicyStatus) => {
    await setPolicyStatus(policy.id, status);
    void mutate({ ...policy, status });
  };

  const onUpdateActionEnabled = async (
    action: Action,
    edit: EditActionRequestBody
  ) => {
    // perform an optimistic update of the action
    const newActions = produce(actions, (draft) => {
      const index = actions.findIndex((a) => a.id === action.id);
      if (edit.enabled !== undefined) draft[index].enabled = edit.enabled;
      if (edit.selectedAdvisoryId !== undefined)
        draft[index].selectedAdvisoryId = edit.selectedAdvisoryId;
    });
    await mutateActions(newActions, false);
    setLoadingPolicy(true);

    const result = await editAction(action.id, edit);
    await mutate(result, true);
    setLoadingPolicy(false);
  };

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
          borderRadius={5}
          shadow="sm"
          justify="space-between"
          borderColor="gray.300"
          borderWidth="thin"
          spacing={8}
        >
          <Stack spacing={5}>
            <Stack spacing={4} bgColor="blue.50" p={3}>
              <Flex
                direction="row"
                justify="space-between"
                spacing={5}
                borderBottomColor="gray.100"
              >
                <Heading size="md">{policy.identity.role}</Heading>
                <HStack align="flex-end" spacing={5}>
                  <Box>
                    <Badge colorScheme="blue">
                      {getEventCountString(policy.eventCount)}
                    </Badge>
                  </Box>
                  <RelativeDateText
                    textAlign="right"
                    date={policy.lastUpdated}
                  />
                </HStack>
              </Flex>
              <HStack>
                {policy.status === "active" ? (
                  <Button
                    leftIcon={<CheckIcon />}
                    colorScheme="blue"
                    variant="outline"
                    size="xs"
                    onClick={() => onSetPolicyStatus("resolved")}
                  >
                    Resolve
                  </Button>
                ) : (
                  <Tooltip hasArrow label="Unresolve">
                    <IconButton
                      size="xs"
                      colorScheme="blue"
                      icon={<CheckIcon />}
                      aria-label="Unresolve"
                      onClick={() => onSetPolicyStatus("active")}
                    />
                  </Tooltip>
                )}
                <Button
                  colorScheme="blue"
                  variant="outline"
                  size="xs"
                  onClick={() =>
                    // TODO: we should direct the user to the specific role URL in the console. To achieve this we need to parse the role name
                    window.open("https://console.aws.amazon.com/iam", "_blank")
                  }
                >
                  View in AWS Console
                </Button>
                <Button
                  colorScheme="blue"
                  variant="outline"
                  size="xs"
                  onClick={onCopy}
                >
                  {hasCopied ? "Policy Copied!" : "Copy Policy"}
                </Button>
              </HStack>
            </Stack>
            <Stack direction="row" wrap="wrap" spacing={3} px={3}>
              <KeyValueBadge label="Role ARN" value={policy.identity.role} />
              <KeyValueBadge label="Account" value={policy.identity.account} />
              <KeyValueBadge label="Token" value={policy.token.name} />
            </Stack>
            <Text px={3}>
              The actions below have been recorded by IAM Zero for this role.
              Confirm your IAM policy by selecting actions and then clicking the
              Copy button on the generated policy.
            </Text>
          </Stack>
          <Stack p={3}>
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
            <Stack {...group}>
              {actions.map((action) => {
                const radio = getRadioProps({ value: action.id });
                return (
                  <ActionDisplay
                    {...radio}
                    key={action.id}
                    action={action}
                    // onMouseOver={() => handleSelectAction(action)}
                    // onMouseOut={() => setSelectedActionId(undefined)}
                    onEditAction={(enabled) =>
                      onUpdateActionEnabled(action, enabled)
                    }
                  />
                );
              })}
            </Stack>
          </Stack>
        </Stack>
      </Stack>
      <Flex
        direction="column"
        py={3}
        position="relative"
        backgroundColor="#011627"
        w="33%"
      >
        <HStack
          justifyContent="center"
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
          <Text
            // apply padding to offset the spinner, so that the text is still centered
            pl="12px"
          >
            Generated Policy
          </Text>
          <Center w="12px" h="12px">
            <Spinner size="xs" display={loadingPolicy ? undefined : "none"} />
          </Center>
        </HStack>
        <Flex flexGrow={1}>
          {/* NB: We deconstruct the JSON so that we can highlight the corresponding
        IAM policy statements when the user selects an action. However this requires 
        a lot of manual styling rather than just dumping the JSON object. There might be
        a more elegant way to achieve this.
         */}
          <Code
            flexGrow={1}
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
            {policy.document.Statement.map((statement) => {
              // TODO: memoize this
              const selectedAction = actions.find(
                (a) => a.id === selectedActionId
              );
              const statementSids = selectedAction?.recommendations?.flatMap(
                (s) => s.AWSPolicy?.Statement.map((s) => s.Sid)
              );
              return (
                <Box
                  key={statement.Sid}
                  position="relative"
                  as="span"
                  display="block"
                  backgroundColor={
                    statementSids?.includes(statement.Sid)
                      ? "whiteAlpha.200"
                      : undefined
                  }
                  px={5}
                >
                  {statementSids?.includes(statement.Sid) && selectedAction && (
                    <Text
                      fontFamily="body"
                      position="absolute"
                      top={1}
                      right={1}
                      color="whiteAlpha.700"
                      fontWeight="bold"
                    >
                      {getAlertTitle(selectedAction)}
                    </Text>
                  )}
                  {"    " +
                    JSON.stringify(statement, null, 2).replace(/\n/g, "\n    ")}
                </Box>
              );
            })}
            <Box as="span" display="block" px={5}>
              {"  "}]
            </Box>
            <Box as="span" display="block" px={5}>
              {`}`}
            </Box>
          </Code>
        </Flex>
        <Flex justify="center" p={5}>
          <Button colorScheme="blue" zIndex="1" onClick={onCopy}>
            {hasCopied ? "Policy Copied!" : "Copy Policy"}
          </Button>
        </Flex>
      </Flex>
    </Flex>
  );
};

interface ActionDisplayProps extends UseRadioProps {
  action: Action;
  onHMouseOver?: () => void;
  onHMouseOut?: () => void;
  onEditAction?: (edit: EditActionRequestBody) => void;
}

const ActionDisplay: React.FC<ActionDisplayProps> = ({
  action,
  onHMouseOut,
  onHMouseOver,
  onEditAction,
  ...rest
}) => {
  const [expanded, setExpanded] = useState(false);

  const { getInputProps, getCheckboxProps } = useRadio(rest);

  const input = getInputProps();
  const checkbox = getCheckboxProps();

  return (
    <Box as="label">
      <input {...input} />
      <Stack
        {...checkbox}
        key={action.id}
        borderWidth="1px"
        p={3}
        boxSizing="border-box"
        borderRadius={5}
        onMouseOver={onHMouseOver}
        onMouseOut={onHMouseOut}
        _checked={{
          borderColor: "transparent",
          boxShadow: "0 0 0 2px #90CDF4",
        }}
        _focus={{
          boxShadow: "outline",
        }}
      >
        <Flex align="center" justify="space-between" flexGrow={1}>
          <Flex align="center">
            <Flex w="100px" justify="center">
              <Checkbox
                isChecked={action.enabled}
                onChange={(e) =>
                  onEditAction?.({
                    enabled: e.target.checked,
                  })
                }
              />
            </Flex>
            <Flex w="200px" justify="flex-end">
              <Text fontWeight="bold">{getAlertTitle(action)}</Text>
            </Flex>
            <Flex w="350px" justify="flex-end">
              <Stack>
                {action.resources.map((resource) => (
                  <Box
                    key={resource.id}
                    as="span"
                    fontSize="xs"
                    borderRadius={5}
                    borderWidth="2px"
                    p={1}
                    backgroundColor="gray.100"
                  >
                    {resource.name}
                  </Box>
                ))}
              </Stack>
            </Flex>
          </Flex>
          <HStack>
            <Select
              value={action.selectedAdvisoryId}
              onChange={(e) =>
                onEditAction?.({
                  selectedAdvisoryId: e.target.value,
                })
              }
              maxW="400px"
            >
              {action.recommendations?.map((advisory) => (
                <option key={advisory.ID} value={advisory.ID}>
                  {advisory.Comment}
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
    </Box>
  );
};

export default PolicyDetails;
