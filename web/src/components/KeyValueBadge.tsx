import { CheckIcon, CopyIcon } from "@chakra-ui/icons";
import { Flex, IconButton, Text, useClipboard } from "@chakra-ui/react";
import React from "react";

interface Props {
  label: string;
  value: string;
}

export const KeyValueBadge: React.FC<Props> = ({ label, value }) => {
  const { hasCopied, onCopy } = useClipboard(value);
  return (
    <Flex
      as="li"
      borderWidth="1px"
      borderRadius={5}
      shadow="sm"
      fontSize="xs"
      align="center"
    >
      <Text
        as="span"
        borderRightWidth="1px"
        p={1}
        h="100%"
        display="flex"
        alignItems="center"
        justifyContent="center"
      >
        {label}
      </Text>
      <Text
        as="span"
        color="blue.500"
        backgroundColor="gray.50"
        fontWeight="medium"
        p={1}
        display="flex"
        alignItems="center"
        justifyContent="center"
      >
        {value}{" "}
        <IconButton
          variant="ghost"
          size="xs"
          icon={hasCopied ? <CheckIcon /> : <CopyIcon />}
          onClick={onCopy}
          aria-label={"Copy " + label}
        />
      </Text>
    </Flex>
  );
};
