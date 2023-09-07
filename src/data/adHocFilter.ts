import { getTable } from './ast';

export class AdHocFilter {
  private _targetTable = '';

  setTargetTable(table: string) {
    this._targetTable = table;
  }

  setTargetTableFromQuery(query: string) {
    this._targetTable = getTable(query);
    if (this._targetTable === '') {
      console.error('Failed to get table from adhoc query.');
      throw new Error('Failed to get table from adhoc query.');
    }
  }

  apply(sql: string, adHocFilters: AdHocVariableFilter[]): string {
    if (sql === '' || !adHocFilters || adHocFilters.length === 0) {
      return sql;
    }
    const filter = adHocFilters[0];

    if (filter.key?.includes('.')) {
      this._targetTable = filter.key.split('.')[0];
    }
    if (this._targetTable === '' || !sql.match(new RegExp(`.*\\b${this._targetTable}\\b.*`, 'gi'))) {
      return sql;
    }

    const filters = adHocFilters
      .filter((filter: AdHocVariableFilter) => {
        const valid = isValid(filter);
        if(!valid) {
          console.error('Invalid adhoc filter will be ignored:', filter);
        }
        return valid;
      })
      .map((f, i) => {
        const key = f.key.includes('.') ? f.key.split('.')[1] : f.key;
        const value = isNaN(Number(f.value)) ? `'${f.value}'` : Number(f.value);
        const condition = i !== adHocFilters.length - 1 ? (f.condition ? f.condition : 'AND') : '';
        const operator = convertOperatorToDatabendOperator(f.operator);
        return ` ${key} ${operator} ${value} ${condition}`;
      })
      .join('');

    if(filters === '') {
      return sql;
    }

    // attach the filters to the query
    const condition = sql.match(/WHERE/i) ? 'AND' : '';
    const re = new RegExp(`("${this._targetTable}") (WHERE)`, 'g');
    sql = sql.replace(re, `$1 WHERE (${filters}) ${condition}`);
    // Semicolons are not required and cause problems when building the SQL
    return sql
  }
}

function isValid(filter: AdHocVariableFilter): boolean {
  return filter.key !== undefined && filter.operator !== undefined && filter.value !== undefined;
}

function convertOperatorToDatabendOperator(operator: AdHocVariableFilterOperator): string {
  if (operator === '=~') {return 'LIKE';}
  if (operator === '!~') {return 'NOT LIKE';}
  return operator;
}

type AdHocVariableFilterOperator = '>' | '<' | '=' | '!=' | '=~' | '!~';

export type AdHocVariableFilter = {
  key: string;
  operator: AdHocVariableFilterOperator;
  value: string;
  condition?: string;
};
