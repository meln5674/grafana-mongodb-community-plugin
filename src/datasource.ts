import { lastValueFrom, Observable } from 'rxjs';
import { 
    DataSourceInstanceSettings,
    DataQueryRequest, 
    DataQueryResponse, 
    MetricFindValue,
    ScopedVars
} from '@grafana/data';
import {
    DataSourceWithBackend, 
    getTemplateSrv,
    frameToMetricFindValue
} from '@grafana/runtime';
import { MongoDBDataSourceOptions, MongoDBQuery, MongoDBQueryType, MongoDBVariableQuery } from './types';

export class DataSource extends DataSourceWithBackend<MongoDBQuery, MongoDBDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MongoDBDataSourceOptions>) {
    super(instanceSettings);
  }

 applyTemplateVariables(query: MongoDBQuery, scopedVars: ScopedVars): Record<string, any> {
    const templateSrv = getTemplateSrv();
    return {
      ...query,
      aggregation: query.aggregation ? templateSrv.replace(query.aggregation, scopedVars, 'json') : ''
    };
  }


  query(request: DataQueryRequest<MongoDBQuery>): Observable<DataQueryResponse> {
      const templateSrv = getTemplateSrv();
      templateSrv.updateTimeRange(request.range);
      return super.query(request);
  }

  async metricFindQuery(query: MongoDBVariableQuery, options?: any): Promise<MetricFindValue[]> {
    const target: Partial<MongoDBQuery> = {
        refId: 'metricFindQuery',
        database: query.database,
        collection: query.collection,
        queryType: MongoDBQueryType.Table,
        timestampField: "",
        timestampFormat: "",
        labelFields: [],
        valueFields: [ query.fieldName ],
        valueFieldTypes: [ query.fieldType ],
        aggregation: query.aggregation,
        autoTimeBound: false,
        autoTimeSort: false,
        schemaInference: false,
        schemaInferenceDepth: 0,
    }

    let dataQuery = {
        ...options,
        targets: [target]
    }
    let dataQueryRequest = dataQuery as DataQueryRequest<MongoDBQuery>

    return lastValueFrom(
        this.query(dataQueryRequest)
    ).then((rsp) => {
        if (rsp.error) {
            throw new Error(rsp.error.message);
        }
        if (rsp.data?.length) {
          return frameToMetricFindValue(rsp.data[0]);
        }
        return [];
    });
  }
}
