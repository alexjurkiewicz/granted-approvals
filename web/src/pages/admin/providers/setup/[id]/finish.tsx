import { ArrowBackIcon, QuestionIcon } from "@chakra-ui/icons";
import {
  Center,
  CircularProgress,
  Code,
  Container,
  HStack,
  IconButton,
  Link as ChakraLink,
  ListItem,
  OrderedList,
  Stack,
  Text,
} from "@chakra-ui/react";
import { Link, Navigate, useMatch } from "react-location";
import ReactMarkdown from "react-markdown";
import {
  CFCode,
  CodeInstruction,
} from "../../../../../components/CodeInstruction";
import { AdminLayout } from "../../../../../components/Layout";
import { useGetProvidersetup } from "../../../../../utils/backend-client/admin/admin";
import { registeredProviders } from "../../../../../utils/providerRegistry";

const Page = () => {
  const {
    params: { id },
  } = useMatch();

  const { data } = useGetProvidersetup(id);

  // used to look up extra details like the name
  const registeredProvider = registeredProviders.find(
    (rp) => rp.type === data?.type
  );

  if (data === undefined) {
    return (
      <AdminLayout>
        <Center borderBottom="1px solid" borderColor="neutrals.200" h="80px">
          <IconButton
            as={Link}
            aria-label="Go back"
            pos="absolute"
            left={4}
            icon={<ArrowBackIcon />}
            rounded="full"
            variant="ghost"
            to="/admin/providers"
          />
          <Text as="h4" textStyle="Heading/H4"></Text>
        </Center>
        <Container
          my={12}
          // This prevents unbounded widths for small screen widths
          minW={{ base: "100%", xl: "container.xl" }}
          overflowX="auto"
        ></Container>
      </AdminLayout>
    );
  }

  const stepsOverview = data?.steps ?? [];

  const completedSteps = stepsOverview.filter((s) => s.complete).length;

  const completedPercentage =
    stepsOverview.length ?? 0 > 0
      ? (completedSteps / stepsOverview.length) * 100
      : 0;

  if (data.status !== "VALIDATION_SUCEEDED") {
    return <Navigate to={`/admin/providers/setup/${data.id}`} />;
  }

  const gdeployCommand =
    "```\n" +
    `gdeploy providers add --id ${data.id} --uses ${data.type}@${
      data.version
    } ${Object.entries(data.configValues)
      .filter(([_, v]) => v != null && v !== "")
      .map(([k, v]) => `--with ${k}=${v}`)
      .join(" ")}` +
    "\n```";

  return (
    <AdminLayout>
      <Stack
        justifyContent={"center"}
        alignItems={"center"}
        spacing={{ base: 1, md: 0 }}
        borderBottom="1px solid"
        borderColor="neutrals.200"
        h="80px"
        py={{ base: 4, md: 0 }}
        flexDirection={{ base: "column", md: "row" }}
      >
        <IconButton
          as={Link}
          aria-label="Go back"
          pos="absolute"
          left={4}
          icon={<ArrowBackIcon />}
          rounded="full"
          variant="ghost"
          to="/admin/providers"
        />
        <Text as="h4" textStyle="Heading/H4">
          {registeredProvider !== undefined &&
            `Setting up the ${registeredProvider.name} provider`}
        </Text>
        {data && (
          <HStack
            spacing={3}
            position={{ md: "absolute", base: "relative" }}
            right={{ md: 4, base: 0 }}
          >
            <Text>
              {completedSteps} of {data.steps.length} steps complete
            </Text>
            <CircularProgress value={completedPercentage} color="#449157" />
          </HStack>
        )}
      </Stack>
      <Container
        my={12}
        // This prevents unbounded widths for small screen widths
        minW={{ base: "100%", xl: "container.xl" }}
        overflowX="auto"
      >
        <Stack
          px={8}
          py={8}
          bg="neutrals.100"
          rounded="md"
          w="100%"
          spacing={8}
        >
          <Text>
            To finish setting up this provider, you need to update your Granted
            deployment configuration.
          </Text>

          <OrderedList color="#757575" spacing={5}>
            <ListItem>
              Ensure that <CFCode>gdeploy</CFCode> is installed on your device.
            </ListItem>
            <ListItem>
              Open a terminal window in the folder containing your{" "}
              <CFCode>granted-deployment.yml</CFCode> file.
            </ListItem>
            <ListItem>
              <Stack>
                <Text>Run the following command:</Text>
                <ReactMarkdown
                  components={{
                    a: (props) => (
                      <Link target="_blank" rel="noreferrer" {...props} />
                    ),
                    p: (props) => (
                      <Text
                        as="span"
                        color="neutrals.600"
                        textStyle={"Body/Small"}
                      >
                        {props.children}
                      </Text>
                    ),
                    code: CodeInstruction,
                  }}
                >
                  {gdeployCommand}
                </ReactMarkdown>
                );
              </Stack>
            </ListItem>
            <ListItem>
              <Stack>
                <Text>Then, update your deployment:</Text>
                <ReactMarkdown
                  components={{
                    a: (props) => (
                      <Link target="_blank" rel="noreferrer" {...props} />
                    ),
                    p: (props) => (
                      <Text
                        as="span"
                        color="neutrals.600"
                        textStyle={"Body/Small"}
                      >
                        {props.children}
                      </Text>
                    ),
                    code: CodeInstruction,
                  }}
                >
                  {"```\ngdeploy update\n```"}
                </ReactMarkdown>
                );
              </Stack>
            </ListItem>
          </OrderedList>
        </Stack>
        {/* <Center mt={5}>
          <Text textStyle={"Body/Small"}>
            Alternatively, you can{" "}
            <ChakraLink>view the YAML configuration</ChakraLink> to set up the
            provider.
          </Text>
        </Center> */}
      </Container>
    </AdminLayout>
  );
};

export default Page;
