import {
  Box,
  Button,
  ButtonGroup,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Stack,
  Text,
} from "@chakra-ui/react";
import React from "react";
import { UnhandledAlert } from "../api-types";
import { getAlertTitle } from "../utils/getAlertTitle";

interface FixAlertModalProps {
  onClose: () => void;
  alert: UnhandledAlert;
}

export const UnhandledAlertModal: React.FC<FixAlertModalProps> = ({
  onClose,
  alert,
}) => {
  return (
    <Modal isOpen={true} onClose={onClose} size="xl">
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>{getAlertTitle(alert)} alert</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <Stack spacing={5}>
            <Text>
              No recommendations currently exist for this alert. We're working
              hard to add some!
            </Text>
            <Box
              fontFamily="mono"
              as="pre"
              fontSize="xs"
              bg="orange.50"
              wordBreak="break-word"
              whiteSpace="pre-wrap"
              p={3}
            >
              {JSON.stringify(alert.event, null, 2)}
            </Box>
          </Stack>
        </ModalBody>

        <ModalFooter>
          <ButtonGroup>
            <Button variant="ghost" onClick={onClose}>
              Close
            </Button>
          </ButtonGroup>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
};
