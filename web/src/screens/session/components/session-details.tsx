import { CodeHighlight, CodeHighlightTabs } from '@mantine/code-highlight'
import { Badge, Button, Flex, type MantineColor, Skeleton, Table, Text } from '@mantine/core'
import { notifications as notify } from '@mantine/notifications'
import {
  IconBrandCSharp,
  IconBrandDebian,
  IconBrandGolang,
  IconBrandJavascript,
  IconBrandNodejs,
  IconBrandPhp,
  IconBrandPython,
  IconBrandWindows,
  IconCup,
  IconDiamond,
  IconExternalLink,
  IconRun,
} from '@tabler/icons-react'
import dayjs from 'dayjs'
import React, { useCallback, useRef } from 'react'
import { useData, UsedStorageKeys, useStorage } from '~/shared'

let count: number = 0

export const SessionDetails = (): React.JSX.Element => {
  console.debug(`🖌 SessionDetails rendering (${++count})`)

  const { session, webHookUrl, sessionLoading } = useData()
  const [selectedShellTab, setSelectedShellTab] = useStorage(0, UsedStorageKeys.SessionDetailsShellTab, 'session')
  const [selectedCodeTab, setSelectedCodeTab] = useStorage(0, UsedStorageKeys.SessionDetailsCodeTab, 'session')
  const testNotifyID = useRef<string | null>(null)

  /** Sends a test request to the generated URL */
  const handleSendTestRequest = useCallback((): void => {
    if (!webHookUrl) {
      return
    }

    if (testNotifyID.current === null) {
      testNotifyID.current = notify.show({
        title: 'Sending request',
        message: 'Please wait...',
        autoClose: false,
        loading: true,
        onClose: () => (testNotifyID.current = null),
      })
    }

    sendTestRequest(new URL(webHookUrl))
      .then(() => {
        if (testNotifyID.current === null) {
          return
        }

        notify.update({
          id: testNotifyID.current,
          title: `Request sent at ${dayjs().format('HH:mm:ss SSS')}`,
          message: 'Check the console for details',
          autoClose: 2000,
          loading: false,
        })
      })
      .catch((error) => {
        if (testNotifyID.current === null) {
          return
        }

        notify.update({
          id: testNotifyID.current,
          title: 'Test request failed',
          message: String(error),
          color: 'orange',
          loading: false,
        })
      })
  }, [webHookUrl])

  return (
    <>
      {!!webHookUrl && (
        <>
          <Text>Here&apos;s your unique URL:</Text>
          <Flex my="md" align="center" justify="space-between" gap="xs" direction={{ base: 'column', lg: 'row' }}>
            <CodeHighlight code={webHookUrl.toString()} language="bash" w="100%" pr="lg" />
            <Button.Group w={{ base: '100%', lg: 'auto' }}>
              <Button
                variant="gradient"
                gradient={{ from: 'cyan', to: 'teal', deg: 90 }}
                leftSection={<IconExternalLink size="1.4em" />}
                component="a"
                href={webHookUrl.toString()}
                target="_blank"
                disabled={sessionLoading}
              >
                Open in a new tab
              </Button>
              <Button
                variant="gradient"
                gradient={{ from: 'teal', to: 'cyan', deg: 90 }}
                leftSection={<IconRun size="1.5em" />}
                onClick={handleSendTestRequest}
                onAuxClick={handleSendTestRequest}
                disabled={sessionLoading}
              >
                Send a request
              </Button>
            </Button.Group>
          </Flex>
          <Text>Send simple POST request (execute next command in your terminal without leaving this page):</Text>
          <CodeHighlightTabs
            code={[
              {
                fileName: 'curl',
                language: 'bash',
                code: `curl -v -X POST --data '{"foo": "bar"}' ${webHookUrl.toString()}`,
                icon: <IconBrandDebian size="1.2em" />,
              },
              {
                fileName: 'wget',
                language: 'bash',
                code: `wget -O- --post-data '{"foo": "bar"}' ${webHookUrl.toString()}`,
                icon: <IconBrandDebian size="1.2em" />,
              },
              {
                fileName: 'HTTPie',
                language: 'bash',
                code: `http POST ${webHookUrl.toString()} foo=bar --verbose`,
                icon: <IconBrandDebian size="1.2em" />,
              },
              {
                fileName: 'get',
                language: 'bash',
                code: `get --data '{"foo": "bar"}' ${webHookUrl.toString()} --method=post --verbose`,
                icon: <IconBrandDebian size="1.2em" />,
              },
              {
                fileName: 'PowerShell',
                language: 'bash',
                code: `Invoke-RestMethod -Uri ${webHookUrl.toString()} -Method POST -Body '{"foo": "bar"}' -Verbose`,
                icon: <IconBrandWindows size="1.2em" />,
              },
            ]}
            onTabChange={(index) => setSelectedShellTab(index)}
            activeTab={selectedShellTab}
            w="100%"
            my="md"
          />
          <Text>Code examples in different languages:</Text>
          <CodeHighlightTabs
            code={[
              {
                fileName: 'JavaScript',
                language: 'javascript',
                code: snippet('js', webHookUrl),
                icon: <IconBrandJavascript size="1.2em" />,
              },
              {
                fileName: 'Node.js',
                language: 'javascript',
                code: snippet('node', webHookUrl),
                icon: <IconBrandNodejs size="1.2em" />,
              },
              {
                fileName: 'Go',
                language: 'go',
                code: snippet('go', webHookUrl),
                icon: <IconBrandGolang size="1.2em" />,
              },
              {
                fileName: 'Java',
                language: 'java',
                code: snippet('java', webHookUrl),
                icon: <IconCup size="1.2em" />,
              },
              {
                fileName: 'Python',
                language: 'python',
                code: snippet('python', webHookUrl),
                icon: <IconBrandPython size="1.2em" />,
              },
              {
                fileName: 'PHP',
                language: 'php',
                code: snippet('php', webHookUrl),
                icon: <IconBrandPhp size="1.2em" />,
              },
              {
                fileName: 'Ruby',
                language: 'ruby',
                code: snippet('ruby', webHookUrl),
                icon: <IconDiamond size="1.2em" />,
              },
              {
                language: 'csharp',
                code: snippet('csharp', webHookUrl),
                icon: <IconBrandCSharp size="1.2em" />,
              },
            ]}
            onTabChange={(index) => setSelectedCodeTab(index)}
            activeTab={selectedCodeTab}
            w="100%"
            my="md"
            defaultExpanded={false}
            withExpandButton
          />
        </>
      )}
      {sessionLoading && [...Array(4)].map((_, i) => <Skeleton height={50} radius="xl" my="md" key={i} />)}
      {!!session && (
        <Table my="md" highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th w={200} ta="right">
                WebHook option
              </Table.Th>
              <Table.Th>Value</Table.Th>
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {!!session.responseCode && (
              <Table.Tr>
                <Table.Td ta="right">Status code</Table.Td>
                <Table.Td>
                  <Badge
                    color={((): MantineColor => {
                      switch (true) {
                        case session.responseCode <= 299:
                          return 'teal'
                        case session.responseCode <= 399:
                          return 'orange'
                        case session.responseCode <= 499:
                          return 'red'
                        default:
                          return 'cyan'
                      }
                    })()}
                  >
                    {session.responseCode}
                  </Badge>
                </Table.Td>
              </Table.Tr>
            )}
            {!!session.responseDelay && (
              <Table.Tr>
                <Table.Td ta="right">Delay</Table.Td>
                <Table.Td>{session.responseDelay} sec</Table.Td>
              </Table.Tr>
            )}
            {!!session.responseHeaders.length && (
              <Table.Tr>
                <Table.Td ta="right">Response headers</Table.Td>
                <Table.Td>
                  <CodeHighlightTabs
                    code={[
                      {
                        code: session.responseHeaders.map(({ name, value }) => `${name}: ${value}`).join('\n'),
                        language: 'bash',
                      },
                    ]}
                    expandCodeLabel="Show all headers"
                    defaultExpanded={false}
                    withHeader={false}
                    withExpandButton
                  />
                </Table.Td>
              </Table.Tr>
            )}
            {!!session.responseBody.length && (
              <Table.Tr>
                <Table.Td ta="right">Response body</Table.Td>
                <Table.Td>
                  <CodeHighlightTabs
                    code={[{ code: String.fromCharCode(...session.responseBody), language: 'json' }]}
                    expandCodeLabel="Show full response"
                    defaultExpanded={false}
                    withHeader={false}
                    withExpandButton
                  />
                </Table.Td>
              </Table.Tr>
            )}
          </Table.Tbody>
        </Table>
      )}
    </>
  )
}

const sendTestRequest = async (url: URL): Promise<Response> => {
  const payload = {
    xhr: 'test',
    now: Math.floor(Date.now() / 1000),
  }

  const methods: Readonly<Array<string>> = ['post', 'put', 'delete', 'patch']

  if (Math.random() > 0.5) {
    url.pathname += '/any/path' // add random path, 50% chance
  }

  url.searchParams.set('test', 'true')
  url.searchParams.set('now', String(payload.now))
  url.hash = '#hash'

  return fetch(
    new Request(url, {
      method: methods[Math.floor(Math.random() * methods.length)].toUpperCase(), // pick random method
      cache: 'no-cache',
      headers: { 'Content-Type': 'application/json', 'X-Test-Header': 'test' },
      body: JSON.stringify(payload),
    })
  )
}

const snippet = (lang: 'js' | 'node' | 'go' | 'java' | 'python' | 'php' | 'ruby' | 'csharp', url: URL): string => {
  switch (lang) {
    case 'js':
      return `fetch('${url.toString()}', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({foo: 'bar'})
})
  .then(res => res.text().then(body => console.log(res.status, body)))
  .catch(console.error)`

    case 'node':
      return `const http = require('http')
const data = JSON.stringify({ foo: 'bar' })

const req = http.request({
  hostname: '${url.hostname}',
  port: ${url.port},
  path: '${url.pathname}',
  headers: { 'Content-Type': 'application/json' },
  method: 'POST',
}, res => {
  res.setEncoding('utf8')
  let body = ''
  res.on('data', chunk => body += chunk)
  res.on('end', () => console.log(res.statusCode, body))
})

req.write(data)
req.end()`

    case 'go':
      return `package main

import (
  "bytes"
  "fmt"
  "io"
  "net/http"
)

func main() {
  resp, err := http.Post( // https://pkg.go.dev/net/http#Post
    "${url.toString()}",
    "application/json",
    bytes.NewBuffer([]byte(\`{"foo": "bar"}\`)),
  )
  if err != nil {
    panic(err)
  }

  defer resp.Body.Close()

  body, err := io.ReadAll(resp.Body)
  if err != nil {
    panic(err)
  }

  fmt.Println(resp.StatusCode, string(body))
}`

    case 'java':
      return `import java.io.*;
import java.net.*;

public class Main {
  public static void main(String[] args) throws Exception {
    URL url = new URL("${url.toString()}");
    HttpURLConnection con = (HttpURLConnection) url.openConnection();

    con.setRequestMethod("POST");
    con.setRequestProperty("Content-Type", "application/json");
    con.setDoOutput(true);
    con.getOutputStream().write("{\\"foo\\":\\"bar\\"}".getBytes());

    try (BufferedReader in = new BufferedReader(new InputStreamReader(con.getInputStream()))) {
      String inputLine;
      while ((inputLine = in.readLine()) != null) System.out.println(inputLine);
    } catch (IOException e) {
      System.out.println(con.getResponseCode());
    }
  }
}`

    case 'python':
      return `import requests

try:
    res = requests.post('${url.toString()}', json={"foo": "bar"})
    print(res.status_code, res.text)
except requests.exceptions.RequestException as e:
    print(e)`

    case 'php':
      return `<?php

require 'vendor/autoload.php';

$client = new GuzzleHttp\\Client(); // https://docs.guzzlephp.org/en/stable/

try {
    $response = $client->post('${url.toString()}', [
        'json' => ['foo' => 'bar']
    ]);

    echo $response->getStatusCode() . ' ' . $response->getBody();
} catch (Exception $e) {
    echo $e->getMessage();
}`

    case 'ruby':
      return `require 'net/http'
require 'uri'

uri = URI.parse("${url.toString()}")
request = Net::HTTP::Post.new(uri, 'Content-Type' => 'application/json')
request.body = '{"foo":"bar"}'

response = Net::HTTP.start(uri.hostname, uri.port) { |http| http.request(request) }
puts response.code, response.body`

    case 'csharp':
      return `using System;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;

class Program {
  static async Task Main() {
    var client = new HttpClient();
    var content = new StringContent("{\\"foo\\":\\"bar\\"}", Encoding.UTF8, "application/json");

    try {
      var response = await client.PostAsync("${url.toString()}", content);
      var body = await response.Content.ReadAsStringAsync();

      Console.WriteLine(response.StatusCode + " " + body);
    } catch (Exception e) {
      Console.WriteLine(e.Message);
    }
  }
}`

    default:
      throw new Error(`Unknown language: ${lang}`)
  }
}
