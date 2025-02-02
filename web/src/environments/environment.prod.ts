/**
 * Copyright 2024 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { PublicKHIExtension } from 'src/app/extensions/public/module';

export const environment = {
  production: true,
  viewerMode: false,
  // For production builds, backend API origin is always matching with the one serving frontend.
  apiBaseUrl: '',
  bugReportUrl:
    'https://github.com/GoogleCloudPlatform/khi/issues/new?template=Blank+issue',
  documentUrl: 'https://github.com/GoogleCloudPlatform/khi',
  pluginModules: [PublicKHIExtension],
  options: {} as Record<string, unknown>,
};
