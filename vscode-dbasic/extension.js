// VS Code extension entry point for DBasic.
// Spawns the dbasic-lsp binary and routes LSP traffic for *.dbas files.
// Set dbasic.lspEnabled = false in settings to use grammar-only mode.
const { workspace } = require('vscode');
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let client;

function activate(context) {
  const config = workspace.getConfiguration('dbasic');
  if (config.get('lspEnabled') === false) {
    return;
  }
  const lspPath = config.get('lspPath') || 'dbasic-lsp';

  const serverOptions = {
    run: { command: lspPath, transport: TransportKind.stdio },
    debug: { command: lspPath, transport: TransportKind.stdio },
  };

  const clientOptions = {
    documentSelector: [{ scheme: 'file', language: 'dbasic' }],
    synchronize: {
      fileEvents: workspace.createFileSystemWatcher('**/*.dbas'),
    },
  };

  client = new LanguageClient(
    'dbasic',
    'DBasic Language Server',
    serverOptions,
    clientOptions
  );

  client.start();
}

function deactivate() {
  if (!client) return undefined;
  return client.stop();
}

module.exports = { activate, deactivate };
