import React from 'react';
import { View, ActivityIndicator, StyleSheet } from 'react-native';
import WebView from 'react-native-webview';
import { useMarkdown } from '../hooks/useCscore';
import { colors } from '../theme';

interface Props {
  content: string;
}

const htmlTemplate = (body: string) => `<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
* { box-sizing: border-box; }
body {
  background: #0a0a0a;
  color: #e8e8e8;
  font-family: -apple-system, Helvetica, sans-serif;
  font-size: 15px;
  line-height: 1.6;
  padding: 16px;
  margin: 0;
}
h1 { font-size: 22px; color: #e8e8e8; margin: 0 0 8px; }
h2 { font-size: 17px; color: #e8e8e8; border-bottom: 1px solid #222; padding-bottom: 8px; margin: 20px 0 10px; }
h3 { font-size: 15px; color: #aaaaaa; margin: 16px 0 8px; }
/* Inline code (not in a block) */
:not(pre) > code {
  font-family: 'Courier New', monospace;
  background: #1a1a1a;
  padding: 2px 5px;
  border-radius: 3px;
  font-size: 13px;
  color: #4a9eff;
}
/* Code blocks — override Chroma's inline background-color with !important */
pre {
  background: #151515 !important;
  border: 1px solid #222;
  border-radius: 6px;
  padding: 12px !important;
  overflow-x: auto;
  margin: 12px 0;
  font-size: 13px;
}
pre > code {
  font-family: 'Courier New', monospace;
  background: transparent !important;
  padding: 0 !important;
  font-size: 13px;
  line-height: 1.5;
}
a { color: #4a9eff; text-decoration: none; }
table { border-collapse: collapse; width: 100%; margin: 12px 0; }
td, th { border: 1px solid #222; padding: 8px 10px; text-align: left; font-size: 14px; }
th { background: #151515; color: #aaaaaa; font-weight: 600; }
ul, ol { padding-left: 20px; margin: 8px 0; }
li { margin: 4px 0; }
p { margin: 8px 0; }
blockquote { border-left: 3px solid #4a9eff; margin: 12px 0; padding: 8px 12px; background: #111111; color: #888888; }
hr { border: none; border-top: 1px solid #222; margin: 16px 0; }
strong { color: #ffffff; }
</style>
</head>
<body>${body}</body>
</html>`;

export function SheetViewer({ content }: Props) {
  const html = useMarkdown(content);

  if (!html) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator color={colors.accent} size="small" />
      </View>
    );
  }

  return (
    <WebView
      source={{ html: htmlTemplate(html) }}
      originWhitelist={['*']}
      style={styles.webview}
    />
  );
}

const styles = StyleSheet.create({
  loading: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: colors.bgPrimary,
  },
  webview: {
    flex: 1,
    backgroundColor: colors.bgPrimary,
  },
});
