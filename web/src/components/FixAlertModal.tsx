import { InfoIcon } from "@chakra-ui/icons";
import {
  Box,
  Button,
  ButtonGroup,
  Heading,
  HStack,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Select,
  Stack,
  Table,
  Tbody,
  Td,
  Text,
  Tooltip,
  Tr,
} from "@chakra-ui/react";
import React, { useState } from "react";
import { ActionWithRecommendations, Recommendation } from "../api-types";
import { getAlertTitle } from "../utils/getAlertTitle";
import { renderStringOrObject } from "../utils/renderStringOrObject";

interface FixAlertModalProps {
  onClose: () => void;
  alert: ActionWithRecommendations;
  onApplyRecommendation?: (recommendationId: string) => void;
}

export const FixAlertModal: React.FC<FixAlertModalProps> = ({
  onClose,
  alert,
  onApplyRecommendation,
}) => {
  const [selectedPolicy, setSelectedPolicy] = useState<Recommendation>(
    alert.recommendations[0]
  );
  const [customisingAction, setCustomisingAction] = useState(false);

  const onSelectPolicy = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const policy = alert.recommendations.find((p) => p.ID === e.target.value);
    if (policy !== undefined) {
      setSelectedPolicy(policy);
      setCustomisingAction(false);
    }
  };

  return (
    <Modal isOpen={true} onClose={onClose} size="xl">
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>Granting {getAlertTitle(alert)}</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <Stack spacing={5}>
            <Stack>
              <Heading size="sm">Parameters</Heading>
              <Table size="sm">
                <Tbody>
                  {Object.entries(alert.event.data.parameters).map(
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
            <Stack>
              <Heading size="sm">Action</Heading>
              {customisingAction === false ? (
                <HStack>
                  <Text>{selectedPolicy?.Comment}</Text>
                  <Button
                    size="xs"
                    variant="outline"
                    onClick={() => setCustomisingAction(true)}
                  >
                    Customise
                  </Button>
                </HStack>
              ) : (
                <Select value={selectedPolicy?.ID} onChange={onSelectPolicy}>
                  {alert.recommendations.map((policy) => (
                    <option key={policy.ID} value={policy.ID}>
                      {policy.Comment}
                    </option>
                  ))}
                </Select>
              )}
            </Stack>
            <Stack>
              <Heading size="sm">Description</Heading>
              {selectedPolicy.Description?.map((description) => (
                <Stack>
                  <Box display="inline" as="span">
                    <Text display="inline">
                      Applying <b>{description.Type}</b> to{" "}
                    </Text>
                    <Tooltip
                      aria-label="Policy ARN tooltip"
                      label={description.AppliedTo}
                    >
                      <HStack as="span" display="inline">
                        <Text fontWeight="bold" display="inline">
                          {description.AppliedTo}
                        </Text>
                        <InfoIcon />
                      </HStack>
                    </Tooltip>
                  </Box>
                  <Box
                    fontFamily="mono"
                    as="pre"
                    fontSize="xs"
                    bg="orange.50"
                    wordBreak="break-word"
                    whiteSpace="pre-wrap"
                    p={3}
                  >
                    {JSON.stringify(description.Policy, null, 2)}
                  </Box>
                </Stack>
              ))}
            </Stack>
          </Stack>
        </ModalBody>

        <ModalFooter>
          <ButtonGroup>
            <Button variant="ghost" onClick={onClose}>
              Cancel
            </Button>
            <Button
              colorScheme="blue"
              mr={3}
              onClick={() => onApplyRecommendation?.(selectedPolicy.ID)}
            >
              Apply
            </Button>
          </ButtonGroup>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};
