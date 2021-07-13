import { AddIcon, PlusSquareIcon } from "@chakra-ui/icons";
import { Stack } from "@chakra-ui/layout";
import {
  Box,
  Button,
  ButtonGroup,
  FormControl,
  FormLabel,
  Input,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Spinner,
  Text,
  useDisclosure,
} from "@chakra-ui/react";
import React, { useState } from "react";
import { createToken, deleteToken, useTokens } from "../api";
import { TokenBox } from "../components/TokenBox";

const Tokens: React.FC = () => {
  const { data, revalidate } = useTokens();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const [tokenName, setTokenName] = useState("");

  const onDeleteToken = async (tokenId: string) => {
    await deleteToken(tokenId);
    await revalidate();
  };

  const onSubmitCreateToken: React.FormEventHandler<HTMLElement> = async (
    event
  ) => {
    event.preventDefault();
    if (tokenName !== "") {
      await createToken(tokenName);
      onClose();
      await revalidate();
    }
  };

  if (data === undefined) return <Spinner />;

  return (
    <>
      {data.tokens.length === 0 ? (
        <Text textAlign="center">
          No tokens!{" "}
          <Button variant="link" onClick={onOpen}>
            Add your first token now to send events to IAM Zero
          </Button>
        </Text>
      ) : (
        <Stack>
          <Box alignSelf="flex-end">
            <Button leftIcon={<AddIcon />} colorScheme="blue" onClick={onOpen}>
              New Token
            </Button>
          </Box>
          <Stack>
            {data.tokens.map((token) => (
              <TokenBox
                key={token.id}
                token={token}
                onDelete={() => onDeleteToken(token.id)}
              />
            ))}
          </Stack>
        </Stack>
      )}
      {isOpen && (
        <Modal isOpen={true} onClose={onClose} size="lg">
          <ModalOverlay />
          <ModalContent as="form" onSubmit={onSubmitCreateToken}>
            <ModalHeader>Create a token</ModalHeader>
            <ModalCloseButton />
            <ModalBody>
              <FormControl>
                <FormLabel>Name</FormLabel>
                <Input
                  placeholder={`A descriptive name (such as "User Details API")`}
                  value={tokenName}
                  onChange={(e) => setTokenName(e.target.value)}
                />
              </FormControl>
            </ModalBody>

            <ModalFooter>
              <ButtonGroup>
                <Button colorScheme="blue" type="submit">
                  Create
                </Button>
              </ButtonGroup>
            </ModalFooter>
          </ModalContent>
        </Modal>
      )}
    </>
  );
};

export default Tokens;
