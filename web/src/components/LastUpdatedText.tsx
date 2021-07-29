import { Text, TextProps } from "@chakra-ui/react";
import { useMemo } from "react";
import { format } from "timeago.js";

interface Props extends TextProps {
  date: Date;
}

/**
 * The `date` prop is memoized
 */
export const RelativeDateText: React.FC<Props> = ({ date, ...rest }) => {
  const renderDate = useMemo(() => format(date), [date]);
  return <Text {...rest}>{renderDate}</Text>;
};
