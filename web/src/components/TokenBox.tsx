import { Flex, Heading } from "@chakra-ui/layout";
import {
  Button,
  ButtonGroup,
  HStack,
  Input,
  InputGroup,
  InputRightElement,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  useDisclosure,
} from "@chakra-ui/react";
import React from "react";
import { Token } from "../api-types";

interface Props {
  token: Token;
  onDelete?: () => void;
}

export const TokenBox: React.FC<Props> = ({ token, onDelete }) => {
  const [show, setShow] = React.useState(false);
  const { isOpen, onOpen, onClose } = useDisclosure();
  const handleClick = () => setShow(!show);

  const handleConfirmDelete = () => {
    onClose();
    onDelete?.();
  };

  return (
    <>
      <Flex
        bg="white"
        p={3}
        borderRadius={5}
        shadow="sm"
        flexDir="row"
        justify="space-between"
        align="center"
        borderColor="gray.300"
        borderWidth="thin"
      >
        <Heading size="sm" flexGrow={1}>
          {token.name}
        </Heading>
        <HStack spacing={10}>
          <InputGroup width="500px">
            <Input
              readOnly
              pr="4.5rem"
              type={show ? "text" : "password"}
              value={token.id}
            />
            <InputRightElement width="4.5rem">
              <Button h="1.75rem" size="sm" onClick={handleClick}>
                {show ? "Hide" : "Show"}
              </Button>
            </InputRightElement>
          </InputGroup>
          <Button colorScheme="red" onClick={onOpen}>
            Delete
          </Button>
        </HStack>
      </Flex>
      {isOpen && (
        <Modal isOpen={true} onClose={onClose} size="lg">
          <ModalOverlay />
          <ModalContent>
            <ModalHeader>
              Are you sure you want to delete this token?
            </ModalHeader>
            <ModalCloseButton />
            <ModalBody>
              Any applications using this token will no longer be able to send
              events to IAM Zero.
            </ModalBody>

            <ModalFooter>
              <ButtonGroup>
                <Button colorScheme="red" onClick={handleConfirmDelete}>
                  I understand, please delete this token
                </Button>
              </ButtonGroup>
            </ModalFooter>
          </ModalContent>
        </Modal>
      )}
    </>
  );
};
