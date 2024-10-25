import React, { useEffect, useState } from 'react'
import { Flex, Divider, Title, Text, Button, Space, Blockquote } from '@mantine/core'
import { CodeHighlight, CodeHighlightTabs } from '@mantine/code-highlight'
import { notifications } from '@mantine/notifications'
import {
  IconExternalLink,
  IconRun,
  IconBrandDebian,
  IconBrandWindows,
  IconBrandJavascript,
  IconBrandNodejs,
  IconBrandGolang,
  IconCup,
  IconBrandPython,
  IconBrandPhp,
  IconDiamond,
  IconBrandCSharp,
  IconInfoCircle,
} from '@tabler/icons-react'
import { sessionToUrl, useLastUsedSID } from '~/shared'

export default function Screen(): React.JSX.Element {
  const [lastUsedSID] = useLastUsedSID()
  const [url, setUrl] = useState<URL>(sessionToUrl(lastUsedSID || '...'))

  useEffect(() => {
    if (lastUsedSID) {
      setUrl(sessionToUrl(lastUsedSID))
    }
  }, [lastUsedSID])

  const handleSendTestRequest = async () => {
    const id = notifications.show({
      title: 'Sending request',
      message: 'Please wait...',
      autoClose: false,
      loading: true,
    })

    try {
      await sendTestRequest(url)

      notifications.update({
        id,
        title: 'Request sent',
        message: 'Check the console for details',
        autoClose: 2000,
        loading: false,
      })
    } catch (error) {
      notifications.update({
        id,
        title: 'Request failed',
        message: String(error),
        color: 'red',
        autoClose: 5000,
        loading: false,
      })
    }
  }

  return (
    <>
      <Title order={3} style={{ fontWeight: 300 }}>
        WebHook Tester allows you to easily test webhooks and other types of HTTP requests
      </Title>
      <Title order={5} c="dimmed" pt={5} style={{ fontWeight: 300 }}>
        Any requests sent to that URL are logged here instantly — you don&apos;t even have to refresh!
      </Title>
      <Divider size="xs" my="lg" variant="dotted" />
      <Text>Here&apos;s your unique URL that was created just now:</Text>
      <Flex my="md" align="center" justify="space-between" gap="xs" direction={{ base: 'column', lg: 'row' }}>
        <CodeHighlight code={url.toString()} language="bash" w="100%" pr="lg" />
        <Button.Group w={{ base: '100%', lg: 'auto' }}>
          <Button
            variant="gradient"
            gradient={{ from: 'cyan', to: 'teal', deg: 90 }}
            leftSection={<IconExternalLink size="1.4em" />}
            component="a"
            href={url.toString()}
            target="_blank"
          >
            Open in a new tab
          </Button>
          <Button
            variant="gradient"
            gradient={{ from: 'teal', to: 'cyan', deg: 90 }}
            leftSection={<IconRun size="1.5em" />}
            onClick={async () => await handleSendTestRequest()}
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
            code: `curl -v -X POST --data '{"foo": "bar"}' ${url.toString()}`,
            icon: <IconBrandDebian size="1.2em" />,
          },
          {
            fileName: 'wget',
            language: 'bash',
            code: `wget -O- --post-data '{"foo": "bar"}' ${url.toString()}`,
            icon: <IconBrandDebian size="1.2em" />,
          },
          {
            fileName: 'HTTPie',
            language: 'bash',
            code: `http POST ${url.toString()} foo=bar --verbose`,
            icon: <IconBrandDebian size="1.2em" />,
          },
          {
            fileName: 'get',
            language: 'bash',
            code: `get --data '{"foo": "bar"}' ${url.toString()} --method=post --verbose`,
            icon: <IconBrandDebian size="1.2em" />,
          },
          {
            fileName: 'PowerShell',
            language: 'bash',
            code: `Invoke-RestMethod -Uri ${url.toString()} -Method POST -Body '{"foo": "bar"}' -Verbose`,
            icon: <IconBrandWindows size="1.2em" />,
          },
        ]}
        w="100%"
        my="md"
      />
      <Text>Code examples in different languages:</Text>
      <CodeHighlightTabs
        code={[
          {
            fileName: 'JavaScript',
            language: 'javascript',
            code: snippet('js', url),
            icon: <IconBrandJavascript size="1.2em" />,
          },
          {
            fileName: 'Node.js',
            language: 'javascript',
            code: snippet('node', url),
            icon: <IconBrandNodejs size="1.2em" />,
          },
          {
            fileName: 'Go',
            language: 'go',
            code: snippet('go', url),
            icon: <IconBrandGolang size="1.2em" />,
          },
          {
            fileName: 'Java',
            language: 'java',
            code: snippet('java', url),
            icon: <IconCup size="1.2em" />,
          },
          {
            fileName: 'Python',
            language: 'python',
            code: snippet('python', url),
            icon: <IconBrandPython size="1.2em" />,
          },
          {
            fileName: 'PHP',
            language: 'php',
            code: snippet('php', url),
            icon: <IconBrandPhp size="1.2em" />,
          },
          {
            fileName: 'Ruby',
            language: 'ruby',
            code: snippet('ruby', url),
            icon: <IconDiamond size="1.2em" />,
          },
          {
            language: 'csharp',
            code: snippet('csharp', url),
            icon: <IconBrandCSharp size="1.2em" />,
          },
        ]}
        w="100%"
        my="md"
      />
      <Space h="xl" />
      <Blockquote color="blue" icon={<IconInfoCircle />}>
        Click &quot;New URL&quot; (in the top right corner) to create a new url with the ability to customize status
        code, response body, etc.
      </Blockquote>
    </>
  )
}

const sendTestRequest = async (url: URL): Promise<Response> => {
  const payload = {
    xhr: 'test',
    now: Math.floor(Date.now() / 1000),
  }

  const methods: Readonly<Array<string>> = ['post', 'put', 'delete', 'patch']

  return fetch(
    new Request(url, {
      method: methods[Math.floor(Math.random() * methods.length)].toUpperCase(), // pick random method
      cache: 'no-cache',
      headers: { 'Content-Type': 'application/json' },
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
