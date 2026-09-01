import type { CodeHighlightTabsCode, CodeHighlightTabsProps } from '@mantine/code-highlight'
import { CodeHighlightTabs } from '@mantine/code-highlight'
import {
  IconBrandGolang,
  IconBrandJavascript,
  IconBrandPhp,
  IconBrandPython,
  IconCup,
  IconDiamond,
  type IconProps,
} from '@tabler/icons-react'
import React, { useMemo } from 'react'
import { L10nKey, useL10n, useStorage } from '~/shared'

const TAB_ICON_PROPS: IconProps = { size: '1.2em' }

export const CodeSnippets = ({
  url,
  ...props
}: Omit<CodeHighlightTabsProps, 'code'> & { url: Readonly<URL> }): React.JSX.Element => {
  const { t } = useL10n()
  const [activeTab, setActiveTab] = useStorage<number>(0, 'session-code-examples-active-tab')

  const tabs = useMemo<Array<CodeHighlightTabsCode>>(() => {
    const asString = url.toString()

    return [
      {
        fileName: 'JavaScript',
        language: 'javascript',
        // language=javascript
        code: `// for both Node.js (≥ v18) and browser environments
fetch('${asString}', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({foo: 'bar'})
})
  .then(res => res.text().then(body => console.log(res.status, body)))
  .catch(console.error)`,
        icon: <IconBrandJavascript {...TAB_ICON_PROPS} />,
      },
      {
        fileName: 'Go',
        language: 'go',
        // language=go
        code: `package main

import (
  "bytes"
  "fmt"
  "io"
  "net/http"
)

func main() {
  resp, err := http.Post( // https://pkg.go.dev/net/http#Post
    "${asString}",
    "application/json",
    bytes.NewBuffer([]byte(\`{"foo": "bar"}\`)),
  )
  if err != nil {
    panic(err)
  }

  defer func() { _ = resp.Body.Close() }()

  body, err := io.ReadAll(resp.Body)
  if err != nil {
    panic(err)
  }

  fmt.Println(resp.StatusCode, string(body))
}`,
        icon: <IconBrandGolang {...TAB_ICON_PROPS} />,
      },
      {
        fileName: 'Python',
        language: 'python',
        // language=python
        code: `import json
import urllib.request

req = urllib.request.Request(
  "${asString}",
  data=json.dumps({"foo": "bar"}).encode(),
  headers={"Content-Type": "application/json"},
  method="POST",
)

try:
  with urllib.request.urlopen(req) as resp:
    print(resp.status, resp.read().decode())
except urllib.error.URLError as e:
  print(e)`,
        icon: <IconBrandPython {...TAB_ICON_PROPS} />,
      },
      {
        fileName: 'PHP',
        language: 'php',
        // language=php
        code: `<?php

$ch = curl_init('${asString}');
curl_setopt_array($ch, [
  CURLOPT_POST => true,
  CURLOPT_POSTFIELDS => json_encode(['foo' => 'bar']),
  CURLOPT_HTTPHEADER => ['Content-Type: application/json'],
  CURLOPT_RETURNTRANSFER => true,
]);

try {
  $body = curl_exec($ch);
  if ($body === false) {
    throw new \\RuntimeException(curl_error($ch));
  }

  echo curl_getinfo($ch, CURLINFO_HTTP_CODE) . ' ' . $body;
} catch (\\Exception $e) {
  echo 'Error: ' . $e->getMessage();
} finally {
  curl_close($ch);
}`,
        icon: <IconBrandPhp {...TAB_ICON_PROPS} />,
      },
      {
        fileName: 'Java',
        language: 'java',
        // language=java
        code: `import java.net.HttpURLConnection;
import java.net.URI;
import java.nio.charset.StandardCharsets;

public class Main {
  public static void main(String[] args) throws Exception {
    var body = "{\\"foo\\":\\"bar\\"}".getBytes(StandardCharsets.UTF_8);
    var conn = (HttpURLConnection) URI.create("${asString}").toURL().openConnection();

    conn.setRequestMethod("POST");
    conn.setRequestProperty("Content-Type", "application/json");
    conn.setDoOutput(true);
    conn.getOutputStream().write(body);

    var status = conn.getResponseCode();
    var stream = status >= 400 ? conn.getErrorStream() : conn.getInputStream();
    var response = stream != null ? new String(stream.readAllBytes(), StandardCharsets.UTF_8) : "";

    System.out.println(status + " " + response);
  }
}`,
        icon: <IconCup {...TAB_ICON_PROPS} />,
      },
      {
        fileName: 'Ruby',
        language: 'ruby',
        // language=ruby
        code: `require 'json'
require 'net/http'
require 'uri'

uri = URI.parse('${asString}')
request = Net::HTTP::Post.new(uri, 'Content-Type' => 'application/json')
request.body = { foo: 'bar' }.to_json

response = Net::HTTP.start(uri.hostname, uri.port, use_ssl: uri.scheme == 'https') do |http|
  http.request(request)
end

puts "#{response.code} #{response.body}"`,
        icon: <IconDiamond {...TAB_ICON_PROPS} />,
      },
      {
        fileName: 'C#',
        language: 'csharp',
        // language=csharp
        code: `using System;
using System.Net.Http;
using System.Net.Http.Json;

using var client = new HttpClient();

try {
  var response = await client.PostAsJsonAsync("${asString}", new { foo = "bar" });
  Console.WriteLine((int)response.StatusCode + " " + await response.Content.ReadAsStringAsync());
} catch (Exception e) {
  Console.WriteLine(e.Message);
}`,
      },
    ]
  }, [url])

  return (
    <CodeHighlightTabs
      withCopyButton
      w="100%"
      copyLabel={t(L10nKey.copy)}
      copiedLabel={t(L10nKey.copied)}
      radius="sm"
      onTabChange={setActiveTab}
      activeTab={Math.min(activeTab ?? 0, tabs.length - 1)}
      withExpandButton
      defaultExpanded={false}
      expandCodeLabel={t(L10nKey.expandCode)}
      collapseCodeLabel={t(L10nKey.collapseCode)}
      {...props}
      code={tabs}
    />
  )
}
