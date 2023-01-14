import React, { ChangeEvent, PureComponent } from 'react';
import {
  FieldSet,
  InlineField,
  InlineFieldRow,
  Input,
  SecretInput,
  TextArea,
  SecretTextArea,
  Field,
  Switch,
} from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MongoDBDataSourceOptions, MongoDBSecureJsonData } from './types';


interface Props extends DataSourcePluginOptionsEditorProps<MongoDBDataSourceOptions> {}

interface State {}

export class ConfigEditor extends PureComponent<Props, State> {
  onURLChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      url: event.target.value,
    } as MongoDBDataSourceOptions;
    onOptionsChange({ ...options, jsonData });
  };
  onTLSCAChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      tlsCa: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onTLSCertificateChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      tlsCertificate: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onTLSChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      tls: event.target.checked,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onTLSServerNameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      tlsServerName: event.target.value,
    };
    onOptionsChange({ ...options, jsonData });
  };

  onUsernameChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const secureJsonData = {
      ...options.secureJsonData,
      username: event.target.value,
    };
    onOptionsChange({ ...options, secureJsonData });
  };
  onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const secureJsonData = {
      ...options.secureJsonData,
      password: event.target.value,
    };
    onOptionsChange({ ...options, secureJsonData });
  };
  onTLSInsecureChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    const jsonData = {
      ...options.jsonData,
      tlsInsecure: event.target.checked,
    };
    onOptionsChange({ ...options, jsonData });
  };
  onTLSCertificateKeyChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    const { onOptionsChange, options } = this.props;
    const secureJsonData = {
      ...options.secureJsonData,
      tlsCertificateKey: event.target.value,
    };
    onOptionsChange({ ...options, secureJsonData });
  };



  onResetCredential = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonFields: {
        ...options.secureJsonFields,
        username: false,
        password: false,
        tlsCertificateKey: false,
      },
    });
  };

  readonly shortWidth = 24;
  readonly longWidth = 56;
  readonly beginCert = "-----BEGIN CERTIFICATE-----";
  readonly endCert = "-----END CERTIFICATE-----";
  readonly beginKey = "-----BEGIN RSA PRIVATE KEY-----";
  readonly endKey = "-----END RSA PRIVATE KEY-----";

  renderCredentials() {
    const { options } = this.props;
    const { secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as MongoDBSecureJsonData;

    return (
      <>
        <InlineFieldRow>
          <InlineField labelWidth={this.shortWidth} label="Username">
            <SecretInput
              width={this.longWidth}
              isConfigured={(secureJsonFields && secureJsonFields.username) as boolean}
              value={secureJsonData.username || ''}
              placeholder="Username"
              onReset={this.onResetCredential}
              onChange={this.onUsernameChange}
            ></SecretInput>
          </InlineField>
          <InlineField label="Password" labelWidth={this.shortWidth}>
            <SecretInput
              width={this.longWidth}
              isConfigured={(secureJsonFields && secureJsonFields.password) as boolean}
              value={secureJsonData.password || ''}
              label="Password"
              placeholder="Password"
              onReset={this.onResetCredential}
              onChange={this.onPasswordChange}
            ></SecretInput>
          </InlineField>
        </InlineFieldRow>
      </>
    )
  }

  renderTls() {
    const { options } = this.props;
    const { jsonData } = options;

    return (
      <>
        <Field label="TLS Enabled">
          <Switch
            value={jsonData.tls || false}
            onChange={this.onTLSChange}
          />
        </Field>
        { jsonData.tls ? this.renderTlsFields() : null }
      </>
    )
  }

  renderTlsFields() {
    const { options } = this.props;
    const { jsonData } = options;

    return (
      <>
        <Field label="Insecure (Skip Verification)">
          <Switch
            value={jsonData.tlsInsecure || false}
            onChange={this.onTLSInsecureChange}
          />
        </Field>
        { jsonData.tlsInsecure ? null : this.renderTlsVerification() }
        { this.renderTlsClient() }
      </>
    )
  }

  renderTlsVerification() {
    const { options } = this.props;
    const { jsonData } = options;
    
    return (
      <> 
        <Field label="TLS Certificate Authority">
          <TextArea
            value={jsonData.tlsCa || ''}
            placeholder={`${this.beginCert}\n...\n${this.endCert}`}
            onChange={this.onTLSCAChange}
            cols={this.longWidth}
          />
        </Field>
        <InlineField
            labelWidth={this.shortWidth}
            label="Expected Server Name"
            tooltip="If your server's certificates are for a different hostname than you use to connect, specify that different hostname here"
            >
          <Input
            width={this.longWidth}
            name="tlsServerName"
            type="text"
            onChange={this.onTLSServerNameChange}
            value={jsonData.tlsServerName || ''}
            placeholder="some.other.hostname"
          ></Input>
        </InlineField>
      </>
    )
  }

  renderTlsClient() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as MongoDBSecureJsonData;

    return (
      <>
        <Field label="TLS Certificate">
          <TextArea
            value={jsonData.tlsCertificate || ''}
            placeholder={`${this.beginCert}\n...\n${this.endCert}`}
            onChange={this.onTLSCertificateChange}
            cols={this.longWidth}
          />
        </Field>
        <br/>
        <Field label="TLS Certificate Key">
          <SecretTextArea
            value={secureJsonData.tlsCertificateKey || ''}
            isConfigured={(secureJsonFields && secureJsonFields.tlsCertificateKey) as boolean}
            placeholder={`${this.beginKey}\n...\n${this.endKey}`}
            onChange={this.onTLSCertificateKeyChange}
            onReset={this.onResetCredential}
            cols={this.longWidth}
          />
        </Field>
      </>
    )
  }

  render() {
    const { options } = this.props;
    const { jsonData } = options;

    return (
      <>
        <FieldSet label="MongoDB Connection" width={400}>
          <InlineField labelWidth={this.shortWidth} label="URL">
            <Input
              width={this.longWidth}
              name="url"
              type="text"
              onChange={this.onURLChange}
              value={jsonData.url || ''}
              placeholder="mongodb[+svc]://hostname:port[,hostname:port][/?key=value]"
            ></Input>
          </InlineField>
          { this.renderCredentials() }
          { this.renderTls() }
        </FieldSet>            
      </>
    );
  }
}
