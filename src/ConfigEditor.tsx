import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
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
      </div>
    );
  }
}
