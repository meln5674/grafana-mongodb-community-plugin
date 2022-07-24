import { lastValueFrom, Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { 
    DataSourceInstanceSettings,
    ScopedVars,
    DataQuery, 
    DataQueryRequest, 
    DataQueryResponse, 
    MetricFindValue 
} from '@grafana/data';
import {
    DataSourceWithBackend, 
    BackendDataSourceResponse,
    FetchResponse,
    getTemplateSrv,
    getBackendSrv,
    toDataQueryResponse
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
    const request: Partial<MongoDBQuery> = {
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
        autoTimeSort: false
    }
    const refId = request.refId || 'variable-query';
    const queries: DataQuery[] = [{ ...request, datasource: this.getRef(), refId }];

    const frame = await lastValueFrom(getBackendSrv().fetch<BackendDataSourceResponse>({
      url: '/api/ds/query',
      method: 'POST',
      data: {
        queries,
      },
      requestId: refId,
    })
    .pipe(
      map((res: FetchResponse<BackendDataSourceResponse>) => {
        const rsp = toDataQueryResponse(res, queries);
        return rsp.data[0];
      })
    ));

    return frame.fields[0].values.buffer.map((value: any) => {
        const metricValue: MetricFindValue = { text: value.toString() };
        return metricValue
    });
  }
}
