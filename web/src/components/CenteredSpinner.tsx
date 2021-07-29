import { Center, Spinner } from "@chakra-ui/react";

export const CenteredSpinner: React.FC = () => {
  return (
    <Center flexGrow={1}>
      <Spinner />
    </Center>
  );
};
