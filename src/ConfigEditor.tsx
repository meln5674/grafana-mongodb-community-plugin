import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms, TextArea, Field, Switch } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { MongoDBDataSourceOptions, MongoDBSecureJsonData } from './types';

const { SecretFormField, FormField } = LegacyForms;

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
      tlsCA: event.target.value,
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
        password: false
      },
      secureJsonData: {
        ...options.secureJsonData,
        username: '',
        password: ''
      },
    });
  };

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as MongoDBSecureJsonData;

    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <FormField
            label="URL"
            labelWidth={6}
            inputWidth={40}
            onChange={this.onURLChange}
            value={jsonData.url || ''}
            placeholder="mongodb[+svc]://hostname:port[,hostname:port][/?key=value]"
          />
        </div>

        <div className="gf-form-inline">
          <div className="gf-form">
            <SecretFormField
              isConfigured={(secureJsonFields && secureJsonFields.username) as boolean}
              value={secureJsonData.username || ''}
              label="Username"
              placeholder="Username"
              labelWidth={6}
              inputWidth={20}
              onReset={this.onResetCredential}
              onChange={this.onUsernameChange}
            />
            <SecretFormField
              isConfigured={(secureJsonFields && secureJsonFields.password) as boolean}
              value={secureJsonData.password || ''}
              label="Password"
              placeholder="Password"
              labelWidth={6}
              inputWidth={20}
              onReset={this.onResetCredential}
              onChange={this.onPasswordChange}
            />

          </div>
        </div>
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
                  placeholder={"-----BEGIN RSA PRIVATE KEY-----\n...\n-----END RSA PRIVATE KEY-----"}
                  onChange={this.onTLSCertificateKeyChange}
                  cols={100}
                />
              </Field>

          </div>
        : null
      }
      </div>
    );
  }
}
