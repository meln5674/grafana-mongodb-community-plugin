import React, { ChangeEvent, PureComponent } from 'react';
import {
  FieldSet,
  InlineField,
  InlineFieldRow,
  Input,
  SecretInput,
  TextArea,
  Field,
  Switch,
  LegacyForms,
} from '@grafana/ui';
const { FormField } = LegacyForms;
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
    };
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

  readonly shortWidth = 15;
  readonly longWidth = 60;

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as MongoDBSecureJsonData;

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
          <InlineFieldRow>
            <InlineField labelWidth={this.shortWidth} label="Username">
              <SecretInput
                width={this.shortWidth}
                isConfigured={(secureJsonFields && secureJsonFields.username) as boolean}
                value={secureJsonData.username || ''}
                placeholder="Username"
                onReset={this.onResetCredential}
                onChange={this.onUsernameChange}
              ></SecretInput>
            </InlineField>
            <InlineField label="Password" labelWidth={this.shortWidth}>
              <SecretInput
                width={this.shortWidth}
                isConfigured={(secureJsonFields && secureJsonFields.password) as boolean}
                value={secureJsonData.password || ''}
                label="Password"
                placeholder="Password"
                onReset={this.onResetCredential}
                onChange={this.onPasswordChange}
              ></SecretInput>
            </InlineField>
          </InlineFieldRow>
        </FieldSet>
        <Field label="TLS Enabled">
            <Switch
              value={jsonData.tls || false}
              onChange={this.onTLSChange}
            />
        </Field>
        { jsonData.tls ?
          <div>
            <Field label="Insecure (Skip Verification)">
                <Switch
                  value={jsonData.tlsInsecure || false}
                  onChange={this.onTLSInsecureChange}
                />
            </Field>
            { !jsonData.tlsInsecure ?
              <> 
              <Field label="TLS Certificate Authority">
                <TextArea
                  value={jsonData.tlsCa || ''}
                  placeholder={"-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"}
                  onChange={this.onTLSCAChange}
                  cols={100}
                />
              </Field>
              <FormField
                label="Expected Server Name"
                labelWidth={16}
                inputWidth={40}
                onChange={this.onTLSServerNameChange}
                value={jsonData.tlsServerName || ''}
                placeholder="some.other.hostname"
                tooltip="If your server's certificates are for a different hostname than you use to connect, specify that different hostname here"
              />
              <br/>
              </>
              : null
            }
              <Field label="TLS Certificate">
                <TextArea
                  value={jsonData.tlsCertificate || ''}
                  placeholder={"-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----"}
                  onChange={this.onTLSCertificateChange}
                  cols={100}
                />
              </Field>
              <br/>
              <Field label="TLS Certificate Key">
                <TextArea
                  value={secureJsonData.tlsCertificateKey || ''}
                  placeholder={secureJsonFields.tlsCertificateKey ? "-----BEGIN RSA PRIVATE KEY-----\n(This field has been set and is hidden)\n-----END RSA PRIVATE KEY-----" : "-----BEGIN RSA PRIVATE KEY-----\n(This field has not been set)\n-----END RSA PRIVATE KEY-----"}
                  onChange={this.onTLSCertificateKeyChange}
                  cols={100}
                />
              </Field>

          </div>
        : null
      }
      </>
    );
  }
}
